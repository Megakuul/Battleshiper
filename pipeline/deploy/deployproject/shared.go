package deployproject

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore/types"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
)

func moveStaticContent(transportCtx context.Context, eventCtx eventcontext.Context) {

}

func updateStaticPageKeys(transportCtx context.Context, eventCtx eventcontext.Context, newStaticPages map[string]string, oldStaticPages map[string]string) error {
	addStaticPageKeys := []types.PutKeyRequestListItem{}
	for key, path := range newStaticPages {
		addStaticPageKeys = append(addStaticPageKeys, types.PutKeyRequestListItem{
			Key:   aws.String(key),
			Value: aws.String(path),
		})
	}

	deleteStaticPageKeys := []types.DeleteKeyRequestListItem{}
	for key, _ := range oldStaticPages {
		if _, exists := newStaticPages[key]; !exists {
			deleteStaticPageKeys = append(deleteStaticPageKeys, types.DeleteKeyRequestListItem{
				Key: aws.String(key),
			})
		}
	}

	storeMetadata, err := eventCtx.CloudfrontCacheClient.DescribeKeyValueStore(transportCtx, &cloudfrontkeyvaluestore.DescribeKeyValueStoreInput{
		KvsARN: aws.String(eventCtx.CloudfrontCacheArn),
	})
	if err != nil {
		return fmt.Errorf("failed to describe cdn store: %v", err)
	}
	_, err = eventCtx.CloudfrontCacheClient.UpdateKeys(transportCtx, &cloudfrontkeyvaluestore.UpdateKeysInput{
		KvsARN:  aws.String(eventCtx.CloudfrontCacheArn),
		Puts:    addStaticPageKeys,
		Deletes: deleteStaticPageKeys,
		IfMatch: storeMetadata.ETag,
	})
	if err != nil {
		return fmt.Errorf("failed to update cdn store keys: %v", err)
	}

	return nil
}
