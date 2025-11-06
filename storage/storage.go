package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
)

type AzureBlobStorage struct {
	client        *azblob.Client
	accountName   string
	containerName string
}

func NewAzureBlobStorage(connStr, storageAccountName, containerName string) *AzureBlobStorage {
	client, err := azblob.NewClientFromConnectionString(connStr, nil)
	if err != nil {
		slog.Error("Error creating service client", slog.String("error", err.Error()))
		panic("couldn't create service client")
	}

	_, err = client.CreateContainer(context.Background(), containerName, nil)
	if err != nil {
		// TODO: Capture the exact error, as this is only really applicable if we got a conflict
		slog.Info("Container already exists", slog.String("container", containerName))
	} else {
		slog.Info("Container created", slog.String("container", containerName))
	}

	return &AzureBlobStorage{
		client:        client,
		accountName:   storageAccountName,
		containerName: containerName,
	}
}

// Takes video data and uploads it to blob storage using the provided name.
// Returns a string containing the URL at which the data can be found.
func (abs *AzureBlobStorage) Upload(fileReader io.Reader, filesize int64, name string) string {
	uploadResponse, err := abs.client.UploadStream(context.Background(), abs.containerName, name, fileReader, nil)

	if err != nil {
		slog.Error("Error uploading blob", slog.String("error", err.Error()))
		return ""
	}

	slog.Info("Uploaded blob", slog.String("name", name), slog.String("etag", string(*uploadResponse.ETag)))
	blobURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", abs.accountName, abs.containerName, name)

	return blobURL
}

// Retrieves the video data by the given name.
// Returns the data as a ReadCloser, as well as the content length and type.
func (abs *AzureBlobStorage) Retrieve(blobName string) (io.ReadCloser, int64, string) {
	downloadResponse, err := abs.client.DownloadStream(context.Background(), abs.containerName, blobName, &blob.DownloadStreamOptions{})
	if err != nil {
		slog.Error("Error downloading blob", slog.String("error", err.Error()))
		return nil, 0, ""
	}

	return downloadResponse.Body, *downloadResponse.ContentLength, *downloadResponse.ContentType
}

func (abs *AzureBlobStorage) Delete(blobName string) error {
	_, err := abs.client.DeleteBlob(context.Background(), abs.containerName, blobName, nil)
	if err != nil {
		slog.Error("Error deleting blob", slog.String("error", err.Error()))
		return err
	}

	slog.Info("Deleted blob", slog.String("name", blobName))
	return nil
}
