package s3

import (
	"errors"
	"time"

	"github.com/redhatinsights/insights-ingress-go/stage"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func getSession() *session.Session {
	return session.Must(session.NewSession())
}

// WithSession returns a stager with a s3 session attached
func WithSession(stager *S3Stager) stage.Stager {
	stager.Sess = getSession()
	return stager
}

// Stage stores the file in s3 and returns a presigned url
func (s *S3Stager) Stage(in *stage.Input) (string, error) {
	uploader := s3manager.NewUploader(s.Sess)
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(in.Key),
		Body:   in.Reader,
	})
	if err != nil {
		return "", errors.New("Failed to upload to s3: " + err.Error())
	}

	client := s3.New(s.Sess)
	req, _ := client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(in.Key),
	})
	url, err := req.Presign(24 * time.Hour)
	if err != nil {
		return "", errors.New("Failed to generate presigned url: " + err.Error())
	}

	return url, nil
}

// Reject moves a payload to the rejected bucket
func (s *S3Stager) Reject(requestID string) error {
	return s.copy(&bucketKey{
		Bucket: s.Bucket,
		Key:    requestID,
	}, s.Rejected)
}

type bucketKey struct {
	Bucket string
	Key    string
}

func (s *S3Stager) copy(from *bucketKey, toBucket string) error {
	src := from.Bucket + "/" + from.Key
	client := s3.New(s.Sess)
	input := &s3.CopyObjectInput{
		Bucket:     aws.String(toBucket),
		CopySource: aws.String(src),
		Key:        aws.String(from.Key),
	}
	_, err := client.CopyObject(input)
	if err != nil {
		return errors.New("Failed to copy from " + src + " to " + toBucket)
	}
	return nil
}