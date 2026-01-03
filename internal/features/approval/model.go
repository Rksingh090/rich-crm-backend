package approval

import (
	"time"

	common_models "go-crm/internal/common/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ApprovalWorkflow defines the rules for approving records in a module
type ApprovalWorkflow struct {
	ID        primitive.ObjectID            `bson:"_id,omitempty" json:"id"`
	ModuleID  string                        `bson:"module_id" json:"module_id"` // The module this workflow applies to
	Name      string                        `bson:"name" json:"name"`
	Active    bool                          `bson:"active" json:"active"`
	Priority  int                           `bson:"priority" json:"priority"` // Evaluation order (0 = highest)
	Criteria  []common_models.RuleCondition `bson:"criteria" json:"criteria"`
	Steps     []ApprovalStep                `bson:"steps" json:"steps"`
	CreatedAt time.Time                     `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time                     `bson:"updated_at" json:"updated_at"`
}

// ApprovalStep defines a single step in the approval process
type ApprovalStep struct {
	ID            string   `bson:"id" json:"id"`                         // Unique ID for the step (e.g., uuid)
	Name          string   `bson:"name" json:"name"`                     // Display name (e.g., "Manager Approval")
	Order         int      `bson:"order" json:"order"`                   // Sequence number
	ApproverRoles []string `bson:"approver_roles" json:"approver_roles"` // Role IDs allowed to approve
	ApproverUsers []string `bson:"approver_users" json:"approver_users"` // User IDs allowed to approve
}
