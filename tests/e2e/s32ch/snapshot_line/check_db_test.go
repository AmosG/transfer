package snapshotline

import (
	"bytes"
	_ "embed"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/stretchr/testify/require"
	"github.com/transferia/transferia/pkg/abstract"
	"github.com/transferia/transferia/pkg/abstract/model"
	chrecipe "github.com/transferia/transferia/pkg/providers/clickhouse/recipe"
	"github.com/transferia/transferia/pkg/providers/s3/s3recipe"
	"github.com/transferia/transferia/tests/helpers"
)

func init() {
	_ = os.Setenv("YC", "1") // to not go to vanga
}

var (
	testBucket = s3recipe.EnvOrDefault("TEST_BUCKET", "barrel")
	target     = *chrecipe.MustTarget(chrecipe.WithInitFile("dump/dump.sql"), chrecipe.WithDatabase("clickhouse_test"))
	//go:embed dump/data.log
	content []byte
	fname   = "data.log"
)

func TestNativeS3(t *testing.T) {
	defer func() {
		require.NoError(t, helpers.CheckConnections(
			helpers.LabeledPort{Label: "CH target Native", Port: target.NativePort},
			helpers.LabeledPort{Label: "CH target HTTP", Port: target.HTTPPort},
		))
	}()

	src := s3recipe.PrepareCfg(t, testBucket, "")
	sess, err := session.NewSession(&aws.Config{
		Endpoint:         aws.String(src.ConnectionConfig.Endpoint),
		Region:           aws.String(src.ConnectionConfig.Region),
		S3ForcePathStyle: aws.Bool(src.ConnectionConfig.S3ForcePathStyle),
		Credentials: credentials.NewStaticCredentials(
			src.ConnectionConfig.AccessKey, string(src.ConnectionConfig.SecretKey), "",
		),
	})
	require.NoError(t, err)

	uploader := s3manager.NewUploader(sess)
	buff := bytes.NewReader(content)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Body:   buff,
		Bucket: aws.String(src.Bucket),
		Key:    aws.String(fname),
	})
	require.NoError(t, err)

	src.TableNamespace = "example"
	src.TableName = "data"
	src.InputFormat = model.ParsingFormatLine
	src.WithDefaults()
	target.WithDefaults()

	transfer := helpers.MakeTransfer("fake", src, &target, abstract.TransferTypeSnapshotOnly)

	helpers.Activate(t, transfer)
	helpers.CheckRowsCount(t, &target, "clickhouse_test", "data", 415)
}
