package internal

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"time"
	"yao/config"
	"yao/db"

	"github.com/super-yaoj/yaoj-core/pkg/problem"
	utils "github.com/super-yaoj/yaoj-utils"
	"github.com/super-yaoj/yaoj-utils/cache"
	"github.com/super-yaoj/yaoj-utils/locks"
)

var (
	ProblemRWLock = locks.NewMappedMultiRWMutex()
	ProblemCache  = cache.NewMemoryCache[*Problem](time.Hour, 100)
)

func ProbGetDir(problem_id int) string {
	return config.Global.DataDir + fmt.Sprint(problem_id)
}

func ProbGetDataZip(problem_id int) string {
	return config.Global.DataDir + fmt.Sprint(problem_id) + ".zip"
}

func ProbGetSampleZip(problem_id int) string {
	return config.Global.DataDir + fmt.Sprint(problem_id) + "_sample.zip"
}

/*
Put problem data in tmp dir first. You should put data zip in tmpdir/1.zip

If the data format is correct, then copy it to the data dir.
*/
func ProbPutData(problem_id int, tmpdir string) error {
	os.Mkdir(path.Join(tmpdir, "1"), os.ModePerm)
	_, err := problem.LoadDump(path.Join(tmpdir, "1.zip"), path.Join(tmpdir, "1"))
	if err != nil {
		return err
	}
	//Success now
	ProblemRWLock.Lock(problem_id)
	defer ProblemRWLock.Unlock(problem_id)
	data_zip := ProbGetDataZip(problem_id)
	data_dir := ProbGetDir(problem_id)
	sample_zip := ProbGetSampleZip(problem_id)

	os.RemoveAll(data_zip)
	os.RemoveAll(data_dir)
	os.RemoveAll(sample_zip)

	os.Rename(path.Join(tmpdir, "1.zip"), data_zip)
	os.Rename(path.Join(tmpdir, "1"), data_dir)

	ProblemCache.Delete(problem_id)
	db.Update("update problems set check_sum=?, allow_down=\"\" where problem_id=?", utils.FileChecksum(data_zip).String(), problem_id)
	return err
}

// Load a problem into memory by problem_id. This function will get the reading lock.
func ProbLoad(problem_id int) *Problem {
	val, ok := ProblemCache.Get(problem_id)
	if !ok {
		ProblemRWLock.RLock(problem_id)
		defer ProblemRWLock.RUnlock(problem_id)
		pro, err := problem.LoadDir(ProbGetDir(problem_id))
		if err != nil {
			log.Print("prload", err)
			return &Problem{}
		}
		ProbSetCache(problem_id, pro)
		ret, _ := ProblemCache.Get(problem_id)
		return ret
	} else {
		return val
	}
}

// Format and set a loaded problem into memory. You should ensure that you have the reading lock before calling this function.
func ProbSetCache(problem_id int, pro problem.Problem) {
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
		HasSample:    utils.FileExists(ProbGetSampleZip(problem_id)),
		TimeLimit:    utils.AtoiDefault(pro.Data().Statement["_tl"], -1),
		MemoryLimit:  utils.AtoiDefault(pro.Data().Statement["_ml"], -1),
	})
}

// You should ensure that you have the writing lock before calling this function.
func ProbModifySample(problem_id int, allow_down string) error {
	ProblemRWLock.Lock(problem_id)
	defer ProblemRWLock.Unlock(problem_id)
	sample_dir := ProbGetSampleZip(problem_id)
	data_dir := ProbGetDir(problem_id)
	os.RemoveAll(sample_dir)
	pro := ProbLoad(problem_id)
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

func ProbHasData(pro *Problem, mode string) bool {
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

func ProbFullScore(problem_id int) float64 {
	pro := ProbLoad(problem_id)
	if pro == nil {
		return 0
	}
	return pro.DataInfo.Fullscore
}
