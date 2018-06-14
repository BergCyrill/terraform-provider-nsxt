/* Copyright © 2018 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: MPL-2.0 */

package nsxt

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	api "github.com/vmware/go-vmware-nsxt"
	"github.com/vmware/go-vmware-nsxt/loadbalancer"
	"log"
	"net/http"
)

func resourceNsxtLbHTTPResponseRewriteRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceNsxtLbHTTPResponseRewriteRuleCreate,
		Read:   resourceNsxtLbHTTPResponseRewriteRuleRead,
		Update: resourceNsxtLbHTTPResponseRewriteRuleUpdate,
		Delete: resourceNsxtLbHTTPRuleDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"revision": getRevisionSchema(),
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Description of this resource",
				Optional:    true,
			},
			"display_name": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The display name of this resource. Defaults to ID if not set",
				Optional:    true,
				Computed:    true,
			},
			"tag": getTagsSchema(),
			"match_strategy": &schema.Schema{
				Type:         schema.TypeString,
				Description:  "Strategy when multiple match conditions are specified in one rule (ANY vs ALL)",
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"ALL", "ANY"}, false),
				Default:      "ALL",
			},
			"request_header_condition":  getLbRuleHTTPHeaderConditionSchema(),
			"response_header_condition": getLbRuleHTTPHeaderConditionSchema(),
			"method_condition":          getLbRuleHTTPRequestMethodConditionSchema(),
			"cookie_condition":          getLbRuleHTTPHeaderConditionSchema(),
			"version_condition":         getLbRuleHTTPVersionConditionSchema(),
			"uri_condition":             getLbRuleHTTPRequestURIConditionSchema(),
			"uri_arguments_condition":   getLbRuleHTTPRequestURIArgumentsConditionSchema(),
			"ip_condition":              getLbRuleIPConditionSchema(),
			"tcp_condition":             getLbRuleTCPConditionSchema(),

			"header_rewrite_action": getLbRuleHeaderRewriteActionSchema(),
		},
	}
}

func getLbRuleHTTPResponseConditionsFromSchema(d *schema.ResourceData) []loadbalancer.LbRuleCondition {
	var conditionList []loadbalancer.LbRuleCondition
	conditions := d.Get("request_header_condition").(*schema.Set).List()
	for _, condition := range conditions {
		data := condition.(map[string]interface{})
		elem := initLbHTTPRuleMatchConditionFromSchema(data, "LbHttpRequestHeaderCondition", true, false)

		conditionList = append(conditionList, elem)
	}

	conditions = d.Get("response_header_condition").(*schema.Set).List()
	for _, condition := range conditions {
		data := condition.(map[string]interface{})
		elem := initLbHTTPRuleMatchConditionFromSchema(data, "LbHttpResponseHeaderCondition", true, false)

		conditionList = append(conditionList, elem)
	}

	conditions = d.Get("cookie_condition").(*schema.Set).List()
	for _, condition := range conditions {
		data := condition.(map[string]interface{})
		elem := initLbHTTPRuleMatchConditionFromSchema(data, "LbHttpRequestCookieCondition", false, true)

		conditionList = append(conditionList, elem)
	}

	conditions = d.Get("method_condition").(*schema.Set).List()
	for _, condition := range conditions {
		data := condition.(map[string]interface{})
		elem := loadbalancer.LbRuleCondition{
			Inverse: data["inverse"].(bool),
			Type_:   "LbHttpRequestMethodCondition",
			Method:  data["method"].(string),
		}

		conditionList = append(conditionList, elem)
	}

	conditions = d.Get("version_condition").(*schema.Set).List()
	for _, condition := range conditions {
		data := condition.(map[string]interface{})
		elem := loadbalancer.LbRuleCondition{
			Inverse: data["inverse"].(bool),
			Type_:   "LbHttpRequestVersionCondition",
			Version: data["version"].(string),
		}

		conditionList = append(conditionList, elem)
	}

	conditions = d.Get("uri_condition").(*schema.Set).List()
	for _, condition := range conditions {
		data := condition.(map[string]interface{})
		elem := initLbHTTPRuleMatchConditionFromSchema(data, "LbHttpRequestUriCondition", false, false)
		elem.Uri = data["uri"].(string)

		conditionList = append(conditionList, elem)
	}

	conditions = d.Get("uri_arguments_condition").(*schema.Set).List()
	for _, condition := range conditions {
		data := condition.(map[string]interface{})
		elem := initLbHTTPRuleMatchConditionFromSchema(data, "LbHttpRequestUriArgumentsCondition", false, false)
		elem.UriArguments = data["uri_arguments"].(string)

		conditionList = append(conditionList, elem)
	}

	conditions = d.Get("ip_condition").(*schema.Set).List()
	for _, condition := range conditions {
		data := condition.(map[string]interface{})
		elem := loadbalancer.LbRuleCondition{
			Inverse:       data["inverse"].(bool),
			Type_:         "LbIpHeaderCondition",
			SourceAddress: data["source_address"].(string),
		}

		conditionList = append(conditionList, elem)
	}

	conditions = d.Get("tcp_condition").(*schema.Set).List()
	for _, condition := range conditions {
		data := condition.(map[string]interface{})
		elem := loadbalancer.LbRuleCondition{
			Inverse:    data["inverse"].(bool),
			Type_:      "LbTcpHeaderCondition",
			SourcePort: data["source_port"].(string),
		}

		conditionList = append(conditionList, elem)
	}

	return conditionList
}

func setLbRuleHTTPResponseConditionsInSchema(d *schema.ResourceData, conditions []loadbalancer.LbRuleCondition) error {
	var requestHeaderConditionList []map[string]interface{}
	var responseHeaderConditionList []map[string]interface{}
	var cookieConditionList []map[string]interface{}
	var methodConditionList []map[string]interface{}
	var versionConditionList []map[string]interface{}
	var uriConditionList []map[string]interface{}
	var uriArgumentsConditionList []map[string]interface{}
	var ipConditionList []map[string]interface{}
	var tcpConditionList []map[string]interface{}

	for _, condition := range conditions {
		elem := make(map[string]interface{})

		if condition.Type_ == "LbHttpRequestHeaderCondition" {
			fillLbHttpRuleHeaderConditionInSchema(elem, condition, false)
			requestHeaderConditionList = append(requestHeaderConditionList, elem)
		}

		if condition.Type_ == "LbHttpResponseHeaderCondition" {
			fillLbHttpRuleHeaderConditionInSchema(elem, condition, false)
			responseHeaderConditionList = append(responseHeaderConditionList, elem)
		}

		if condition.Type_ == "LbHttpRequestCookieCondition" {
			fillLbHttpRuleHeaderConditionInSchema(elem, condition, true)
			cookieConditionList = append(cookieConditionList, elem)
		}

		if condition.Type_ == "LbHttpRequestMethodCondition" {
			elem["method"] = condition.Method
			elem["inverse"] = condition.Inverse
			methodConditionList = append(methodConditionList, elem)
		}

		if condition.Type_ == "LbHttpRequestVersionCondition" {
			elem["version"] = condition.Version
			elem["inverse"] = condition.Inverse
			versionConditionList = append(versionConditionList, elem)
		}

		if condition.Type_ == "LbHttpRequestUriCondition" {
			elem["uri"] = condition.Uri
			elem["inverse"] = condition.Inverse
			elem["match_type"] = condition.MatchType
			elem["case_sensitive"] = *condition.CaseSensitive
			uriConditionList = append(uriConditionList, elem)
		}

		if condition.Type_ == "LbHttpRequestUriArgumentsCondition" {
			elem["uri_arguments"] = condition.UriArguments
			elem["inverse"] = condition.Inverse
			elem["match_type"] = condition.MatchType
			elem["case_sensitive"] = *condition.CaseSensitive
			uriArgumentsConditionList = append(uriArgumentsConditionList, elem)
		}

		if condition.Type_ == "LbIpHeaderCondition" {
			elem["source_address"] = condition.SourceAddress
			elem["inverse"] = condition.Inverse
			ipConditionList = append(ipConditionList, elem)
		}

		if condition.Type_ == "LbTcpHeaderCondition" {
			elem["source_port"] = condition.SourcePort
			elem["inverse"] = condition.Inverse
			tcpConditionList = append(tcpConditionList, elem)
		}

		err := d.Set("request_header_condition", requestHeaderConditionList)
		if err != nil {
			return err
		}

		err = d.Set("response_header_condition", responseHeaderConditionList)
		if err != nil {
			return err
		}

		err = d.Set("cookie_condition", cookieConditionList)
		if err != nil {
			return err
		}

		err = d.Set("method_condition", methodConditionList)
		if err != nil {
			return err
		}

		err = d.Set("version_condition", versionConditionList)
		if err != nil {
			return err
		}

		err = d.Set("uri_condition", uriConditionList)
		if err != nil {
			return err
		}

		err = d.Set("uri_arguments_condition", uriArgumentsConditionList)
		if err != nil {
			return err
		}

		err = d.Set("ip_condition", ipConditionList)
		if err != nil {
			return err
		}

		err = d.Set("tcp_condition", tcpConditionList)
		if err != nil {
			return err
		}

	}
	return nil
}

func getLbRuleResponseRewriteActionsFromSchema(d *schema.ResourceData) []loadbalancer.LbRuleAction {
	var actionList []loadbalancer.LbRuleAction
	actions := d.Get("header_rewrite_action").(*schema.Set).List()
	for _, action := range actions {
		data := action.(map[string]interface{})
		elem := loadbalancer.LbRuleAction{
			Type_:       "LbHttpResponseHeaderRewriteAction",
			HeaderName:  data["name"].(string),
			HeaderValue: data["value"].(string),
		}

		actionList = append(actionList, elem)
	}

	return actionList
}

func setLbRuleResponseRewriteActionsInSchema(d *schema.ResourceData, actions []loadbalancer.LbRuleAction) error {
	var headerActionList []map[string]string

	for _, action := range actions {
		elem := make(map[string]string)
		if action.Type_ == "LbHttpResponseHeaderRewriteAction" {
			elem["name"] = action.HeaderName
			elem["value"] = action.HeaderValue
			headerActionList = append(headerActionList, elem)
		}

		err := d.Set("header_rewrite_action", headerActionList)
		if err != nil {
			return err
		}
	}

	return nil
}

func resourceNsxtLbHTTPResponseRewriteRuleCreate(d *schema.ResourceData, m interface{}) error {
	nsxClient := m.(*api.APIClient)
	description := d.Get("description").(string)
	displayName := d.Get("display_name").(string)
	tags := getTagsFromSchema(d)
	matchConditions := getLbRuleHTTPResponseConditionsFromSchema(d)
	actions := getLbRuleResponseRewriteActionsFromSchema(d)
	matchStrategy := d.Get("match_strategy").(string)
	phase := "HTTP_RESPONSE_REWRITE"

	lbRule := loadbalancer.LbRule{
		Description:     description,
		DisplayName:     displayName,
		Tags:            tags,
		Actions:         actions,
		MatchConditions: matchConditions,
		MatchStrategy:   matchStrategy,
		Phase:           phase,
	}

	lbRule, resp, err := nsxClient.ServicesApi.CreateLoadBalancerRule(nsxClient.Context, lbRule)

	if err != nil {
		return fmt.Errorf("Error during LoadBalancerRule create: %v", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Unexpected status returned during LoadBalancerRule create: %v", resp.StatusCode)
	}
	d.SetId(lbRule.Id)

	return resourceNsxtLbHTTPResponseRewriteRuleRead(d, m)
}

func resourceNsxtLbHTTPResponseRewriteRuleRead(d *schema.ResourceData, m interface{}) error {
	nsxClient := m.(*api.APIClient)
	id := d.Id()
	if id == "" {
		return fmt.Errorf("Error obtaining logical object id")
	}

	lbRule, resp, err := nsxClient.ServicesApi.ReadLoadBalancerRule(nsxClient.Context, id)
	if err != nil {
		return fmt.Errorf("Error during LoadBalancerRule read: %v", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("[DEBUG] LoadBalancerRule %s not found", id)
		d.SetId("")
		return nil
	}
	d.Set("revision", lbRule.Revision)
	d.Set("description", lbRule.Description)
	d.Set("display_name", lbRule.DisplayName)
	setTagsInSchema(d, lbRule.Tags)
	setLbRuleHTTPResponseConditionsInSchema(d, lbRule.MatchConditions)
	d.Set("match_strategy", lbRule.MatchStrategy)
	err = setLbRuleResponseRewriteActionsInSchema(d, lbRule.Actions)
	if err != nil {
		log.Printf("[DEBUG] Failed to set action in LoadBalancerRule %d: %v", id, err)
		d.SetId("")
		return nil
	}

	return nil
}

func resourceNsxtLbHTTPResponseRewriteRuleUpdate(d *schema.ResourceData, m interface{}) error {
	nsxClient := m.(*api.APIClient)
	id := d.Id()
	if id == "" {
		return fmt.Errorf("Error obtaining logical object id")
	}

	revision := int32(d.Get("revision").(int))
	description := d.Get("description").(string)
	displayName := d.Get("display_name").(string)
	tags := getTagsFromSchema(d)
	matchConditions := getLbRuleHTTPResponseConditionsFromSchema(d)
	actions := getLbRuleResponseRewriteActionsFromSchema(d)
	matchStrategy := d.Get("match_strategy").(string)
	phase := "HTTP_RESPONSE_REWRITE"

	lbRule := loadbalancer.LbRule{
		Revision:        revision,
		Description:     description,
		DisplayName:     displayName,
		MatchStrategy:   matchStrategy,
		Phase:           phase,
		Actions:         actions,
		MatchConditions: matchConditions,
		Tags:            tags,
	}

	lbRule, resp, err := nsxClient.ServicesApi.UpdateLoadBalancerRule(nsxClient.Context, id, lbRule)

	if err != nil || resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("Error during LoadBalancerRule update: %v", err)
	}

	return resourceNsxtLbHTTPResponseRewriteRuleRead(d, m)
}
