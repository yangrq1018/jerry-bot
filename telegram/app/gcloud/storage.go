package gcloud

import (
	"bytes"
	"cloud.google.com/go/storage"
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"time"
)

// global unique storage key, site ownership endorsed
const (
	storageBucket = "yangruoqi.site"
)

func LoadObject(object string, data interface{}) error {
	ctx := context.Background()
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer storageClient.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	rc, err := storageClient.Bucket(storageBucket).Object(object).NewReader(ctx)
	if err == storage.ErrObjectNotExist {
		return err
	}
	defer rc.Close()
	err = json.NewDecoder(rc).Decode(data)
	if err != nil {
		return err
	}
	return nil
}

// SaveObject overwrite object on Google Cloud Storage with key object
func SaveObject(object string, o interface{}) error {
	ctx := context.Background()
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer storageClient.Close()

	// Open local file.
	f := bytes.NewBuffer(nil)
	err = json.NewEncoder(f).Encode(&o)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	// Upload an object with storage.Writer.
	wc := storageClient.Bucket(storageBucket).Object(object).NewWriter(ctx)
	if _, err = io.Copy(wc, f); err != nil {
		return fmt.Errorf("io.Copy: %v", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %v", err)
	}
	log.Printf("Cloud: blob with key %q uploaded", object)
	return nil
}
