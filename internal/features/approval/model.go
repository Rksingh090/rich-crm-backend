package approval

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ApprovalProcess defines the rules for approving records in a module
type RuleCondition struct {
	Field    string `json:"field" bson:"field"`
	Operator string `json:"operator" bson:"operator"`
	Value    any    `json:"value" bson:"value"`
}

type ApprovalProcess struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID  primitive.ObjectID `bson:"tenant_id" json:"tenant_id"`
	ModuleID  string             `bson:"module_id" json:"module_id"` // The module this process applies to
	Name      string             `bson:"name" json:"name"`
	Active    bool               `bson:"active" json:"active"`
	Priority  int                `bson:"priority" json:"priority"` // Evaluation order (0 = highest)
	Criteria  []RuleCondition    `bson:"criteria" json:"criteria"`
	Steps     []ApprovalStep     `bson:"steps" json:"steps"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// ApprovalAction defines an action to be executed (e.g. send email, webhook)
type ApprovalAction struct {
	ID     string         `bson:"id" json:"id"`         // Unique ID
	Type   string         `bson:"type" json:"type"`     // e.g. "send_email", "webhook", "update_record"
	Config map[string]any `bson:"config" json:"config"` // Action specific config
	Order  int            `bson:"order" json:"order"`   // Execution order
}

// ApprovalStep defines a single step in the approval process
type ApprovalStep struct {
	ID            string           `bson:"id" json:"id"`                         // Unique ID for the step (e.g., uuid)
	Name          string           `bson:"name" json:"name"`                     // Display name (e.g., "Manager Approval")
	Order         int              `bson:"order" json:"order"`                   // Sequence number
	ApproverRoles []string         `bson:"approver_roles" json:"approver_roles"` // Role IDs allowed to approve
	ApproverUsers []string         `bson:"approver_users" json:"approver_users"` // User IDs allowed to approve
	BeforeActions []ApprovalAction `bson:"before_actions" json:"before_actions"` // Actions to run before this step becomes active
	AfterActions  []ApprovalAction `bson:"after_actions" json:"after_actions"`   // Actions to run after this step is approved
}
