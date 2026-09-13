package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
)

// AzureBlob implements Storage backed by Azure Blob Storage.
type AzureBlob struct {
	client      *azblob.Client
	svcClient   *service.Client
	accountName string
	container   string
}

// NewAzureBlob creates a new Azure Blob Storage-backed store.
// accountName is the storage account name (AZURE_STORAGE_ACCOUNT).
// container is the blob container name (AZURE_STORAGE_CONTAINER, default "dicom").
// Authentication uses DefaultAzureCredential — managed identity in Azure Container Apps,
// AZURE_CLIENT_ID/AZURE_CLIENT_SECRET/AZURE_TENANT_ID env vars locally.
func NewAzureBlob(ctx context.Context, accountName, container string) (*AzureBlob, error) {
	if accountName == "" {
		return nil, errors.New("AZURE_STORAGE_ACCOUNT is required when STORAGE_MODE=azure")
	}
	if container == "" {
		container = "dicom"
	}

	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("azure credential: %w", err)
	}

	client, err := azblob.NewClient(serviceURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure blob client: %w", err)
	}

	svcClient, err := service.NewClient(serviceURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure service client: %w", err)
	}

	return &AzureBlob{
		client:      client,
		svcClient:   svcClient,
		accountName: accountName,
		container:   container,
	}, nil
}

func (a *AzureBlob) GenerateUploadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	now := time.Now().UTC()
	udk, err := a.svcClient.GetUserDelegationCredential(ctx, service.KeyInfo{
		Start:  to.Ptr(now.Add(-10 * time.Second).Format(sas.TimeFormat)),
		Expiry: to.Ptr(now.Add(expiry).Format(sas.TimeFormat)),
	}, nil)
	if err != nil {
		return "", fmt.Errorf("azure user delegation key: %w", err)
	}

	sasParams, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     now.Add(-10 * time.Second),
		ExpiryTime:    now.Add(expiry),
		Permissions:   to.Ptr(sas.BlobPermissions{Write: true, Create: true}).String(),
		ContainerName: a.container,
		BlobName:      key,
	}.SignWithUserDelegation(udk)
	if err != nil {
		return "", fmt.Errorf("azure sign upload url: %w", err)
	}

	return fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s?%s",
		a.accountName, a.container, key, sasParams.Encode()), nil
}

func (a *AzureBlob) GenerateDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	now := time.Now().UTC()
	udk, err := a.svcClient.GetUserDelegationCredential(ctx, service.KeyInfo{
		Start:  to.Ptr(now.Add(-10 * time.Second).Format(sas.TimeFormat)),
		Expiry: to.Ptr(now.Add(expiry).Format(sas.TimeFormat)),
	}, nil)
	if err != nil {
		return "", fmt.Errorf("azure user delegation key: %w", err)
	}

	sasParams, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     now.Add(-10 * time.Second),
		ExpiryTime:    now.Add(expiry),
		Permissions:   to.Ptr(sas.BlobPermissions{Read: true}).String(),
		ContainerName: a.container,
		BlobName:      key,
	}.SignWithUserDelegation(udk)
	if err != nil {
		return "", fmt.Errorf("azure sign download url: %w", err)
	}

	return fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s?%s",
		a.accountName, a.container, key, sasParams.Encode()), nil
}

func (a *AzureBlob) Store(ctx context.Context, key string, r io.Reader) error {
	_, err := a.client.UploadStream(ctx, a.container, key, r, nil)
	if err != nil {
		return fmt.Errorf("azure blob upload: %w", err)
	}
	return nil
}

func (a *AzureBlob) Retrieve(ctx context.Context, key string) (io.ReadCloser, error) {
	resp, err := a.client.DownloadStream(ctx, a.container, key, nil)
	if err != nil {
		return nil, fmt.Errorf("azure blob download: %w", err)
	}
	return resp.Body, nil
}

func (a *AzureBlob) List(ctx context.Context, prefix string) ([]string, error) {
	p := prefix + "/"
	pager := a.client.NewListBlobsFlatPager(a.container, &azblob.ListBlobsFlatOptions{
		Prefix: &p,
	})

	var keys []string
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("azure blob list: %w", err)
		}
		for _, item := range page.Segment.BlobItems {
			if item.Name != nil && *item.Name != "" && !strings.HasSuffix(*item.Name, "/") {
				keys = append(keys, *item.Name)
			}
		}
	}
	return keys, nil
}

func (a *AzureBlob) Delete(ctx context.Context, key string) error {
	_, err := a.client.DeleteBlob(ctx, a.container, key, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return nil // idempotent
		}
		return fmt.Errorf("azure blob delete: %w", err)
	}
	return nil
}

func (a *AzureBlob) Move(ctx context.Context, srcKey, dstKey string) error {
	// Azure Blob has no native rename — copy then delete.
	srcURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s",
		a.accountName, a.container, srcKey)
	dst := a.svcClient.NewContainerClient(a.container).NewBlockBlobClient(dstKey)
	_, err := dst.StartCopyFromURL(ctx, srcURL, nil)
	if err != nil {
		return fmt.Errorf("azure blob copy: %w", err)
	}
	return a.Delete(ctx, srcKey)
}

func (a *AzureBlob) Size(ctx context.Context, key string) (int64, error) {
	props, err := a.svcClient.NewContainerClient(a.container).
		NewBlobClient(key).
		GetProperties(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("azure blob properties: %w", err)
	}
	if props.ContentLength == nil {
		return 0, nil
	}
	return *props.ContentLength, nil
}

// KeyToPath returns empty for cloud storage (files are not on the local filesystem).
func (a *AzureBlob) KeyToPath(_ string) string {
	return ""
}
