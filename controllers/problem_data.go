package controllers

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"sync"
	"yao/libs"

	"github.com/sshwy/yaoj-core/pkg/problem"
	"github.com/sshwy/yaoj-core/pkg/utils"
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

func PRGetKey(problem_id int) string {
	return fmt.Sprintf("problem_%d", problem_id)
}

/*
Put problem data in tmp dir first. You should put data zip in tmpdir/1.zip

If the data format is correct, then copy it to the data dir.
*/
func PRPutData(problem_id int, tmpdir string) error {
	os.Mkdir(tmpdir+"/1", os.ModePerm)
	_, err := problem.LoadDump(tmpdir+"/1.zip", tmpdir+"/1")
	if err != nil {
		return err
	}
	//Success now
	ProblemWLock(problem_id)
	defer ProblemWUnlock(problem_id)
	data_zip := PRGetDataZip(problem_id)
	data_dir := PRGetDir(problem_id)
	sample_zip := PRGetSampleZip(problem_id)

	os.RemoveAll(data_zip)
	os.RemoveAll(data_dir)
	os.RemoveAll(sample_zip)

	os.Rename(tmpdir+"/1.zip", data_zip)
	os.Rename(tmpdir+"/1", data_dir)

	libs.CacheMap.Delete(PRGetKey(problem_id))
	libs.DBUpdate("update problems set check_sum=?, allow_down=\"\" where problem_id=?", utils.FileChecksum(data_zip).String(), problem_id)
	return err
}

//Load a problem into memory by problem_id. This function will get the reading lock.
func PRLoad(problem_id int) *Problem {
	val, ok := libs.CacheMap.Get(PRGetKey(problem_id))
	if !ok {
		ProblemRLock(problem_id)
		defer ProblemRUnlock(problem_id)
		pro, err := problem.LoadDir(PRGetDir(problem_id))
		if err != nil {
			return &Problem{}
		}
		PRSetCache(problem_id, pro)
		ret, _ := libs.CacheMap.Get(PRGetKey(problem_id))
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
	libs.CacheMap.Set(PRGetKey(problem_id), &Problem{
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
	ProblemWLock(problem_id)
	defer ProblemWUnlock(problem_id)
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

var problemRWLock sync.Map

func getProblemRWLock(problem_id int) *sync.RWMutex {
	lock, ok := problemRWLock.Load(problem_id)
	if !ok {
		lock = new(sync.RWMutex)
		problemRWLock.Store(problem_id, lock)
	}
	return lock.(*sync.RWMutex)
}

func ProblemRLock(problem_id int) {
	getProblemRWLock(problem_id).RLock()
}

func ProblemRUnlock(problem_id int) {
	getProblemRWLock(problem_id).RUnlock()
}

func ProblemWLock(problem_id int) {
	getProblemRWLock(problem_id).Lock()
}

func ProblemWUnlock(problem_id int) {
	getProblemRWLock(problem_id).Unlock()
}
