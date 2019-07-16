package signalfx

import (
	"encoding/json"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	chart "github.com/signalfx/signalfx-go/chart"
)

func eventFeedChartResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the chart",
			},
			"program_text": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Signalflow program text for the chart. More info at \"https://developers.signalfx.com/docs/signalflow-overview\"",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the chart (Optional)",
			},
			"time_range": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				Description:   "Seconds to display in the visualization. This is a rolling range from the current time. Example: 3600 = `-1h`",
				ConflictsWith: []string{"start_time", "end_time"},
			},
			"start_time": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				Description:   "Seconds since epoch to start the visualization",
				ConflictsWith: []string{"time_range"},
			},
			"end_time": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				Description:   "Seconds since epoch to end the visualization",
				ConflictsWith: []string{"time_range"},
			},
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL of the chart",
			},
		},

		Create: eventFeedChartCreate,
		Read:   eventFeedChartRead,
		Update: eventFeedChartUpdate,
		Delete: eventFeedChartDelete,
		Exists: chartExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

/*
  Use Resource object to construct json payload in order to create an event feed chart
*/
func getPayloadEventFeedChart(d *schema.ResourceData) *chart.CreateUpdateChartRequest {
	var timeOptions *chart.TimeDisplayOptions
	if val, ok := d.GetOk("time_range"); ok {
		timeOptions = &chart.TimeDisplayOptions{
			Range: int64(val.(int) * 1000),
			Type:  "relative",
		}
	}
	if val, ok := d.GetOk("start_time"); ok {
		timeOptions = &chart.TimeDisplayOptions{
			Start: int64(val.(int) * 1000),
			Type:  "absolute",
		}
		if val, ok := d.GetOk("end_time"); ok {
			timeOptions.End = int64(val.(int) * 1000)
		}
	}

	return &chart.CreateUpdateChartRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		ProgramText: d.Get("program_text").(string),
		Options: &chart.Options{
			Time: timeOptions,
			Type: "Event",
		},
	}
}

func eventFeedChartCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadEventFeedChart(d)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create Event Feed Chart Payload: %s", string(debugOutput))

	c, err := config.Client.CreateChart(payload)
	if err != nil {
		return err
	}
	// Since things worked, set the URL and move on
	appURL, err := buildAppURL(config.CustomAppURL, CHART_APP_PATH+c.Id)
	if err != nil {
		return err
	}
	d.Set("url", appURL)
	if err := d.Set("url", appURL); err != nil {
		return err
	}
	d.SetId(c.Id)
	return eventfeedchartAPIToTF(d, c)
}

func eventfeedchartAPIToTF(d *schema.ResourceData, c *chart.Chart) error {
	debugOutput, _ := json.Marshal(c)
	log.Printf("[DEBUG] SignalFx: Got Event Feed Chart to enState: %s", string(debugOutput))

	if err := d.Set("name", c.Name); err != nil {
		return err
	}
	if err := d.Set("description", c.Description); err != nil {
		return err
	}
	if err := d.Set("program_text", c.ProgramText); err != nil {
		return err
	}

	return nil
}

func eventFeedChartRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	c, err := config.Client.GetChart(d.Id())
	if err != nil {
		return err
	}

	return eventfeedchartAPIToTF(d, c)
}

func eventFeedChartUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload := getPayloadEventFeedChart(d)
	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Event Feed Chart Payload: %s", string(debugOutput))

	c, err := config.Client.UpdateChart(d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Update Event Feed Chart Response: %v", c)

	d.SetId(c.Id)
	return eventfeedchartAPIToTF(d, c)
}

func eventFeedChartDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteChart(d.Id())
}
