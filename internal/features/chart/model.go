package chart

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChartType string
type AggregationType string

const (
	ChartTypeBar         ChartType = "bar"
	ChartTypeLine        ChartType = "line"
	ChartTypePie         ChartType = "pie"
	ChartTypeDonut       ChartType = "donut"
	ChartTypeArea        ChartType = "area"
	ChartTypeFunnel      ChartType = "funnel"
	ChartTypeScatter     ChartType = "scatter"
	ChartTypeRadar       ChartType = "radar"
	ChartTypeStackedBar  ChartType = "stacked_bar"
	ChartTypeStackedArea ChartType = "stacked_area"
)

const (
	AggregationTypeCount AggregationType = "count"
	AggregationTypeSum   AggregationType = "sum"
	AggregationTypeAvg   AggregationType = "avg"
	AggregationTypeMin   AggregationType = "min"
	AggregationTypeMax   AggregationType = "max"
)

type Chart struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name            string             `json:"name" bson:"name"`
	Description     string             `json:"description" bson:"description"`
	ModuleID        string             `json:"module_id" bson:"module_id"`
	ChartType       ChartType          `json:"chart_type" bson:"chart_type"`
	XAxisField      string             `json:"x_axis_field" bson:"x_axis_field"`
	YAxisField      string             `json:"y_axis_field,omitempty" bson:"y_axis_field,omitempty"`
	AggregationType AggregationType    `json:"aggregation_type" bson:"aggregation_type"`
	CreatedAt       time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at" bson:"updated_at"`
}
