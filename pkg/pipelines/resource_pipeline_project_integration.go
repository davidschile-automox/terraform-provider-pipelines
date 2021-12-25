package pipelines

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// ProjectIntegration GET {{ host }}/pipelines/api/v1/projectintegrations/{{projectIntegrationId}}

type ProjectIntegration struct {
	//Project                   string          `json:"project"`
	Name                  string           `json:"name"`
	ProjectId             int              `json:"projectId"`
	MasterIntegrationId   int              `json:"masterIntegrationId"`
	MasterIntegrationName string           `json:"masterIntegrationName"`
	FormJSONValues        []FormJSONValues `json:"formJSONValues"`
	Environments          []string         `json:"environments,omitempty"`
	IsInternal            bool             `json:"isInternal,omitempty"`
	ID                    int              `json:"id,omitempty"`
}

type FormJSONValues struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

const projectIntegrationsUrl = "pipelines/api/v1/projectintegrations"

// func verifyProjectIntegration(id string, request *resty.Request) (*resty.Response, error) {
// 	return request.Head(pipelinesSourcesUrl + id)
// }

func pipelineProjectIntegrationResource() *schema.Resource {

	var projectIntegrationSchema = map[string]*schema.Schema{
		"name": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the project integration. Should be prefixed with the project key",
		},
		"project_id": {
			Type:         schema.TypeInt,
			Required:     true,
			ValidateFunc: validation.IntAtLeast(0),
			Description:  "Id of the project.",
		},
		"master_integration_id": {
			Type:         schema.TypeInt,
			Required:     true,
			ValidateFunc: validation.IntAtLeast(0),
			Description:  "The Id of the master integration.",
		},
		"master_integration_name": {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the master integration.",
		},
		"form_json_values": {
			Type:     schema.TypeList,
			Required: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"label": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "Key or label of the input property.",
					},
					"value": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "Value of the input property.",
					},
				},
			},
			Description: "multiple objects with the values for the integration.",
		},
		"environments": {
			Type:     schema.TypeList,
			Optional: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Description: "In a project, an array of environment names in which this pipeline source will be.",
		},
		"is_internal": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Set this as false to create a Pipelines integration.",
		},
	}

	var unpackFormJSONValues = func(d *ResourceData, key string) []FormJSONValues {
		var formJSONValues []FormJSONValues
		keyValues := d.Get(key).([]interface{})
		for _, keyValue := range keyValues {
			idx := keyValue.(map[string]interface{})
			formJSONValue := FormJSONValues{
				Label: idx["label"].(string),
				Value: idx["value"].(string),
			}
			formJSONValues = append(formJSONValues, formJSONValue)
		}
		return formJSONValues
	}

	var packFormJSONValues = func(d *schema.ResourceData, schemaKey string, formJSONValues []FormJSONValues) []error {
		setValue := mkLens(d)
		var keyValues []interface{}
		for _, idx := range formJSONValues {
			keyValue := map[string]interface{}{
				"label": idx.Label,
				"value": idx.Value,
			}
			keyValues = append(keyValues, keyValue)
		}
		errors := setValue(schemaKey, keyValues)
		return errors
	}

	var unpackProjectIntegration = func(data *schema.ResourceData) (ProjectIntegration, error) {
		d := &ResourceData{data}

		projectIntegration := ProjectIntegration{
			Name:                  d.getString("name"),
			ProjectId:             d.getInt("project_id"),
			MasterIntegrationId:   d.getInt("master_integration_id"),
			MasterIntegrationName: d.getString("master_integration_name"),
			Environments:          d.getList("environments"),
			IsInternal:            d.getBool("is_internal"),
			FormJSONValues:        unpackFormJSONValues(d, "form_json_values"),
		}
		return projectIntegration, nil
	}

	var packProjectIntegration = func(d *schema.ResourceData, projectIntegration ProjectIntegration) diag.Diagnostics {
		var errors []error
		setValue := mkLens(d)

		errors = setValue("project_id", projectIntegration.ProjectId)
		setValue("name", projectIntegration.Name)
		setValue("project_id", projectIntegration.ProjectId)
		setValue("master_integration_id", projectIntegration.MasterIntegrationId)
		setValue("master_integration_id", projectIntegration.MasterIntegrationName)
		setValue("environments", projectIntegration.Environments)
		setValue("is_internal", projectIntegration.IsInternal)
		errors = packFormJSONValues(d, "form_json_values", projectIntegration.FormJSONValues)

		if len(errors) > 0 {
			return diag.Errorf("failed to pack project integration %q", errors)
		}

		return nil
	}

	var readProjectIntegration = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		projectIntegration := ProjectIntegration{}
		_, err := m.(*resty.Client).R().
			SetResult(&projectIntegration).
			Get(projectIntegrationsUrl + "/" + data.Id())
		if err != nil {
			return diag.FromErr(err)
		}
		return packProjectIntegration(data, projectIntegration)
	}

	var createProjectIntegration = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		log.Printf("[DEBUG] createProjectIntegration")
		log.Printf("[TRACE] %+v\n", data)

		projectIntegration, err := unpackProjectIntegration(data)
		if err != nil {
			return diag.FromErr(err)
		}

		resp, err := m.(*resty.Client).R().SetBody(projectIntegration).Post(projectIntegrationsUrl)
		if err != nil {
			return diag.FromErr(err)
		}
		var result ProjectIntegration
		err = json.Unmarshal(resp.Body(), &result)
		if err != nil {
			return diag.FromErr(err)
		}
		data.SetId(strconv.Itoa(result.ID))

		return readProjectIntegration(ctx, data, m)
	}

	var updateProjectIntegration = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		log.Printf("[DEBUG] updateProjectIntegration")
		log.Printf("[TRACE] %+v\n", data)

		projectIntegration, err := unpackProjectIntegration(data)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = m.(*resty.Client).R().
			SetBody(projectIntegration).
			Put(projectIntegrationsUrl + "/" + data.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		return readProjectIntegration(ctx, data, m)
	}

	var deleteProjectIntegration = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		log.Printf("[DEBUG] deleteProjectIntegration")
		log.Printf("[TRACE] %+v\n", data)

		resp, err := m.(*resty.Client).R().
			Delete(projectIntegrationsUrl + "/" + data.Id())

		if err != nil && resp.StatusCode() == http.StatusNotFound {
			data.SetId("")
			return diag.FromErr(err)
		}

		return nil
	}

	return &schema.Resource{
		SchemaVersion: 1,
		CreateContext: createProjectIntegration,
		ReadContext:   readProjectIntegration,
		UpdateContext: updateProjectIntegration,
		DeleteContext: deleteProjectIntegration,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema:      projectIntegrationSchema,
		Description: "Provides an Artifactory Pipeline Source resource.",
	}
}
