package azurerm

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/riviera/azure"
)

func resourceArmLoadbalancerFrontEndIpConfig() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmLoadbalancerFrontEndIpConfigCreate,
		Read:   resourceArmLoadbalancerFrontEndIpConfigRead,
		Update: resourceArmLoadbalancerFrontEndIpConfigCreate,
		Delete: resourceArmLoadbalancerFrontEndIpConfigDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": {
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"loadbalancer_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"subnet_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"private_ip_address": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"public_ip_address_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"private_ip_address_allocation": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateLoadbalancerPrivateIpAddressAllocation,
			},
		},
	}
}

func resourceArmLoadbalancerFrontEndIpConfigCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	lbClient := client.loadBalancerClient

	loadBalancer, exists, err := retrieveLoadbalancerById(d.Get("loadbalancer_id").(string), meta)
	if err != nil {
		return err
	}
	if !exists {
		d.SetId("")
		return nil
	}

	resGroup := d.Get("resource_group_name").(string)
	loadBalancerName := *loadBalancer.Name
	newLb := mergeLoadbalancerConfig(loadBalancer)
	newLb.Properties = &network.LoadBalancerPropertiesFormat{
		FrontendIPConfigurations: expandAzureRmLoadbalancerFrontendIpConfigurations(d),
	}

	_, err = lbClient.CreateOrUpdate(resGroup, loadBalancerName, newLb, make(chan struct{}))
	if err != nil {
		return err
	}

	read, err := lbClient.Get(resGroup, loadBalancerName, "")
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Loadbalancer %s (resource group %s) ID", loadBalancerName, resGroup)
	}

	d.SetId(*read.ID)

	log.Printf("[DEBUG] Waiting for LoadBalancer (%s) to become available", loadBalancerName)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted", "Updating"},
		Target:  []string{"Succeeded"},
		Refresh: loadbalancerStateRefreshFunc(client, resGroup, loadBalancerName),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Loadbalancer (%s) to become available: %s", loadBalancerName, err)
	}

	return resourceArmLoadbalancerFrontEndIpConfigRead(d, meta)
}

func expandAzureRmLoadbalancerFrontendIpConfigurations(d *schema.ResourceData) *[]network.FrontendIPConfiguration {
	frontEndConfigs := make([]network.FrontendIPConfiguration, 0)

	properties := &network.FrontendIPConfigurationPropertiesFormat{}

	if v := d.Get("public_ip_address_id").(string); v != "" {
		properties.PublicIPAddress = &network.PublicIPAddress{
			ID: &v,
		}
	}

	//for _, configRaw := range configs {
	//	data := configRaw.(map[string]interface{})
	//
	//	private_ip_allocation_method := data["private_ip_address_allocation"].(string)
	//	properties := network.FrontendIPConfigurationPropertiesFormat{
	//		PrivateIPAllocationMethod: network.IPAllocationMethod(private_ip_allocation_method),
	//	}
	//
	//	if v := data["private_ip_address"].(string); v != "" {
	//		properties.PrivateIPAddress = &v
	//	}
	//
	//
	//
	//	if v := data["subnet_id"].(string); v != "" {
	//		properties.Subnet = &network.Subnet{
	//			ID: &v,
	//		}
	//	}
	//
	//	name := data["name"].(string)
	//	frontEndConfig := network.FrontendIPConfiguration{
	//		Name:       &name,
	//		Properties: &properties,
	//	}
	//
	//
	//}

	frontEndConfig := network.FrontendIPConfiguration{
		Name:       azure.String(d.Get("name").(string)),
		Properties: properties,
	}

	frontEndConfigs = append(frontEndConfigs, frontEndConfig)

	return &frontEndConfigs
}

func resourceArmLoadbalancerFrontEndIpConfigRead(d *schema.ResourceData, meta interface{}) error {
	loadBalancer, exists, err := retrieveLoadbalancerById(d.Id(), meta)
	if err != nil {
		return err
	}
	if !exists {
		d.SetId("")
		log.Printf("[INFO] Loadbalancer %q not found. Refreshing from state", d.Get("name").(string))
		return nil
	}

	d.Set("name", loadBalancer.Properties.FrontendIPConfigurations[0].Name)

	flattenAndSetTags(d, loadBalancer.Tags)

	return nil
}

func resourceArmLoadbalancerFrontEndIpConfigDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func validateLoadbalancerPrivateIpAddressAllocation(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	allocations := map[string]bool{
		"static":  true,
		"dynamic": true,
	}

	if !allocations[value] {
		errors = append(errors, fmt.Errorf("Loadbalancer Allocations can only be Static or Dynamic"))
	}
	return
}
