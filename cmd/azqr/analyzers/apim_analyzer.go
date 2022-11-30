package analyzers

import (
	"context"
	"log"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/apimanagement/armapimanagement"
)

type ApiManagementAnalyzer struct {
	diagnosticsSettings DiagnosticsSettings
	subscriptionId      string
	ctx                 context.Context
	cred                azcore.TokenCredential
}

func NewApiManagementAnalyzer(subscriptionId string, ctx context.Context, cred azcore.TokenCredential) *ApiManagementAnalyzer {
	diagnosticsSettings, _ := NewDiagnosticsSettings(cred, ctx)
	analyzer := ApiManagementAnalyzer{
		diagnosticsSettings: *diagnosticsSettings,
		subscriptionId:      subscriptionId,
		ctx:                 ctx,
		cred:                cred,
	}
	return &analyzer
}

func (a ApiManagementAnalyzer) Review(resourceGroupName string) ([]AzureServiceResult, error) {
	log.Printf("Analyzing API Management Services in Resource Group %s", resourceGroupName)

	services, err := a.listServices(resourceGroupName)
	if err != nil {
		return nil, err
	}
	results := []AzureServiceResult{}
	for _, s := range services {
		hasDiagnostics, err := a.diagnosticsSettings.HasDiagnostics(*s.ID)
		if err != nil {
			return nil, err
		}

		sku := string(*s.SKU.Name)
		sla := "99.95%"
		if strings.Contains(sku, "Premium") && (len(s.Zones) > 0 || len(s.Properties.AdditionalLocations) > 0) {
			sla = "99.99%"
		}

		results = append(results, AzureServiceResult{
			SubscriptionId:     a.subscriptionId,
			ResourceGroup:      resourceGroupName,
			ServiceName:        *s.Name,
			Sku:                sku,
			Sla:                sla,
			Type:               *s.Type,
			AvailabilityZones:  len(s.Zones) > 0,
			PrivateEndpoints:   len(s.Properties.PrivateEndpointConnections) > 0,
			DiagnosticSettings: hasDiagnostics,
			CAFNaming:          strings.HasPrefix(*s.Name, "apim"),
		})
	}
	return results, nil
}

func (a ApiManagementAnalyzer) listServices(resourceGroupName string) ([]*armapimanagement.ServiceResource, error) {

	servicesClient, err := armapimanagement.NewServiceClient(a.subscriptionId, a.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := servicesClient.NewListByResourceGroupPager(resourceGroupName, nil)

	services := make([]*armapimanagement.ServiceResource, 0)
	for pager.More() {
		resp, err := pager.NextPage(a.ctx)
		if err != nil {
			return nil, err
		}
		services = append(services, resp.Value...)
	}
	return services, nil
}