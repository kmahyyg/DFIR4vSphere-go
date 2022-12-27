package res_storage

import "os"

var (
	ResultSaver = &OutputSaver{}
)

type OutputSaver struct {
	destFile *os.File
}

func (osaver *OutputSaver) Init(outputName string) (err error) {
	osaver.destFile, err = os.Create(outputName)
	if err != nil {
		return err
	}
	return nil
}

func (osaver *OutputSaver) Cleanup() (err error) {
	return nil
}

type JsonOutputFormatter struct {
}
