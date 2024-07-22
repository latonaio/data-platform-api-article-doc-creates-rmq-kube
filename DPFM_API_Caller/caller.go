package dpfm_api_caller

import (
	"context"
	"crypto/sha256"
	dpfm_api_input_reader "data-platform-api-article-doc-creates-rmq-kube/DPFM_API_Input_Formatter"
	dpfm_api_output_formatter "data-platform-api-article-doc-creates-rmq-kube/DPFM_API_Output_Formatter"
	"data-platform-api-article-doc-creates-rmq-kube/config"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	rabbitmq "github.com/latonaio/rabbitmq-golang-client-for-data-platform"
	"os"
	"time"

	"github.com/latonaio/golang-logging-library-for-data-platform/logger"
)

type DPFMAPICaller struct {
	ctx  context.Context
	conf *config.Conf
	rmq  *rabbitmq.RabbitmqClient
}

func NewDPFMAPICaller(
	conf *config.Conf, rmq *rabbitmq.RabbitmqClient,
) *DPFMAPICaller {
	return &DPFMAPICaller{
		ctx:  context.Background(),
		conf: conf,
		rmq:  rmq,
	}
}

func generateFile(
	directoryPath string,
	fileName string,
	fileExtension string,
	docData string,
	log *logger.Logger,
) error {
	dec, err := base64.StdEncoding.DecodeString(docData)
	if err != nil {
		return err
	}

	err = createDirectory(directoryPath)
	if err != nil {
		return err
	}

	fileFullPath := fmt.Sprintf("%s/%s.%s",
		directoryPath,
		fileName,
		fileExtension,
	)

	f, err := os.Create(fileFullPath)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Fatal("failed to close file", err)
		}
	}(f)

	if _, err := f.Write(dec); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}

	return nil
}

func createDirectory(directoryPath string) error {
	err := os.MkdirAll(directoryPath, 0777)
	if err != nil {
		return err
	}
	return nil
}

func encodeToStringSha256(s string) string {
	r := sha256.Sum256([]byte(s))
	return hex.EncodeToString(r[:])
}

func (c *DPFMAPICaller) AsyncDocCreates(
	input *dpfm_api_input_reader.SDC,
	output *dpfm_api_output_formatter.SDC,
	log *logger.Logger,
	errs *[]error,
	conf *config.Conf,
) (interface{}, []error) {
	var fileExtension = input.Article.HeaderDoc.FileExtension
	var docData = input.DocData

	var combinedString = fmt.Sprintf(
		"%v%v",
		time.Now().Unix(),
		input.Article.HeaderDoc.DocVersionID,
	)

	directoryPath := fmt.Sprintf("%s",
		conf.MountPath,
	)

	var dockId = fmt.Sprintf("%s", encodeToStringSha256(combinedString))

	err := generateFile(directoryPath, dockId, fileExtension, docData, log)
	if err != nil {
		*errs = append(*errs, err)
		return nil, nil
	}

	response := c.createSqlProcess(input, output, &dpfm_api_output_formatter.HeaderDoc{
		Article:                  input.Article.Article,
		DocType:                  input.Article.HeaderDoc.DocType,
		DocVersionID:             input.Article.HeaderDoc.DocVersionID,
		DocID:                    &dockId,
		FileExtension:            input.Article.HeaderDoc.FileExtension,
		FileName:                 input.Article.HeaderDoc.FileName,
		FilePath:                 directoryPath,
		DocIssuerBusinessPartner: input.Article.HeaderDoc.DocIssuerBusinessPartner,
	}, errs, log)

	return response, nil
}
