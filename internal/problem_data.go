package internal

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"time"
	"yao/libs"

	"github.com/k0kubun/pp"
	"github.com/super-yaoj/yaoj-core/pkg/problem"
	"github.com/super-yaoj/yaoj-core/pkg/utils"
)

var (
	ProblemRWLock = libs.NewMappedMultiRWMutex()
	ProblemCache  = libs.NewMemoryCache(time.Hour, 100)
)

func PRGetDir(problem_id int) string {
	return libs.DataDir + fmt.Sprint(problem_id)
}

func PRGetDataZip(problem_id int) string {
	return libs.DataDir + fmt.Sprint(problem_id) + ".zip"
}

func PRGetSampleZip(problem_id int) string {
	return libs.DataDir + fmt.Sprint(problem_id) + "_sample.zip"
}


/*
Put problem data in tmp dir first. You should put data zip in tmpdir/1.zip

If the data format is correct, then copy it to the data dir.
*/
func PRPutData(problem_id int, tmpdir string) error {
	os.Mkdir(path.Join(tmpdir, "1"), os.ModePerm)
	_, err := problem.LoadDump(path.Join(tmpdir, "1.zip"), path.Join(tmpdir, "1"))
	if err != nil {
		return err
	}
	//Success now
	ProblemRWLock.Lock(problem_id)
	defer ProblemRWLock.Unlock(problem_id)
	data_zip := PRGetDataZip(problem_id)
	data_dir := PRGetDir(problem_id)
	sample_zip := PRGetSampleZip(problem_id)

	os.RemoveAll(data_zip)
	os.RemoveAll(data_dir)
	os.RemoveAll(sample_zip)

	os.Rename(path.Join(tmpdir, "1.zip"), data_zip)
	os.Rename(path.Join(tmpdir, "1"), data_dir)

	ProblemCache.Delete(problem_id)
	libs.DBUpdate("update problems set check_sum=?, allow_down=\"\" where problem_id=?", utils.FileChecksum(data_zip).String(), problem_id)
	return err
}

//Load a problem into memory by problem_id. This function will get the reading lock.
func PRLoad(problem_id int) *Problem {
	val, ok := ProblemCache.Get(problem_id)
	if !ok {
		ProblemRWLock.RLock(problem_id)
		defer ProblemRWLock.RUnlock(problem_id)
		pro, err := problem.LoadDir(PRGetDir(problem_id))
		if err != nil {
			log.Print("prload", err)
			return &Problem{}
		}
		pp.Print(pro)
		PRSetCache(problem_id, pro)
		ret, _ := ProblemCache.Get(problem_id)
		return ret.(*Problem)
	} else {
		return val.(*Problem)
	}
}

//Format and set a loaded problem into memory. You should ensure that you have the reading lock before calling this function.
func PRSetCache(problem_id int, pro problem.Problem) {
	stmts := []Statement{}
	for key, val := range pro.Data().Statement {
		if key[0] != '_' {
			stmts = append(stmts, Statement{key, val})
		}
	}
	ProblemCache.Set(problem_id, &Problem{
		Id:           problem_id,
		Statement_zh: string(pro.Stmt("zh")),
		Statement_en: string(pro.Stmt("en")),
		Tutorial_zh:  string(pro.Tutr("zh")),
		Tutorial_en:  string(pro.Tutr("en")),
		DataInfo:     pro.DataInfo(),
		Statements:   stmts,
		SubmConfig:   pro.Data().Submission,
		HasSample:    libs.FileExists(PRGetSampleZip(problem_id)),
		TimeLimit:    libs.AtoiDefault(pro.Data().Statement["_tl"], -1),
		MemoryLimit:  libs.AtoiDefault(pro.Data().Statement["_ml"], -1),
	})
}

//You should ensure that you have the writing lock before calling this function.
func PRModifySample(problem_id int, allow_down string) error {
	ProblemRWLock.Lock(problem_id)
	defer ProblemRWLock.Unlock(problem_id)
	sample_dir := PRGetSampleZip(problem_id)
	data_dir := PRGetDir(problem_id)
	os.RemoveAll(sample_dir)
	pro := PRLoad(problem_id)
	zipfile, _ := os.Create(sample_dir)
	writer := zip.NewWriter(zipfile)

	success := false
	for key, val := range pro.Statements {
		if key < len(allow_down) && allow_down[key] == '1' {
			file, err := os.Open(data_dir + "/" + val.Path)
			if err != nil {
				continue
			}
			defer file.Close()
			f, err := writer.Create(val.Name)
			if err != nil {
				continue
			}
			_, err = io.Copy(f, file)
			if err != nil {
				writer.Close()
				zipfile.Close()
				os.RemoveAll(sample_dir)
				pro.HasSample = false
				return err
			}
			success = true
		}
	}
	writer.Close()
	zipfile.Close()
	if !success {
		os.RemoveAll(sample_dir)
		pro.HasSample = false
	} else {
		pro.HasSample = true
	}
	return nil
}

func totalTests(test *problem.TestdataInfo) int {
	ret := 0
	for _, sub := range test.Subtasks {
		ret += len(sub.Tests)
	}
	return ret
}

func PRHasData(pro *Problem, mode string) bool {
	switch mode {
	case "pretest":
		return totalTests(&pro.DataInfo.Pretest) > 0
	case "tests":
		return totalTests(&pro.DataInfo.TestdataInfo) > 0
	case "extra":
		return totalTests(&pro.DataInfo.Extra) > 0
	}
	return false
}

func PRFullScore(problem_id int) float64 {
	pro := PRLoad(problem_id)
	if pro == nil {
		return 0
	}
	return pro.DataInfo.Fullscore
}