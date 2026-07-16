// sudoapi: Model catalog.

package service_model_catalog

import (
	"github.com/google/wire"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func ProvideModelCatalogService(
	channelSvc *service.ChannelService,
	pricingSvc *service.PricingService,
	metadataSvc *MetadataService,
	endpointConfigSvc *EndpointConfigService,
) *ModelCatalogService {
	return NewModelCatalogService(channelSvc, pricingSvc, metadataSvc, endpointConfigSvc)
}

func ProvideModelCatalogMetadataService(
	repo MetadataRepository,
	channelSvc *service.ChannelService,
	pricingSvc *service.PricingService,
) *MetadataService {
	return NewModelCatalogMetadataService(repo, channelSvc, pricingSvc)
}

// ProviderSet is the Wire provider set for all services
var ProviderSet = wire.NewSet(
	NewEndpointConfigService,
	ProvideModelCatalogMetadataService,
	ProvideModelCatalogService,
	wire.Bind(new(service.ModelCatalogCacheInvalidator), new(*ModelCatalogService)),
)
