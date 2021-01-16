package mongo

import "errors"

var (
	ErrFailedReadInputFile = errors.New("could not read the input file")
)

func ErrWriteFileToDatabase(fileName string) error {
	return errors.New("could not write to GridFs for " + fileName)
}
