package domain

import "time"

const (
	ActionLoginSuccess         = "LOGIN_SUCCESS"
	ActionLoginFailure         = "LOGIN_FAILURE"
	ActionLogout               = "LOGOUT"
	ActionPatientCreate        = "patient.create"
	ActionRegisterCreate       = "register.create"
	ActionRegisterCheckIn      = "register.check-in"
	ActionRegisterComplete     = "register.complete"
	ActionRegisterCancel       = "register.cancel"
	ActionDiagnoseCreate       = "diagnose.create"
	ActionDiagnoseCommit       = "diagnose.commit"
	ActionDiagnoseAbate        = "diagnose.abate"
	ActionOrderCreate          = "order.create"
	ActionOrderCommit          = "order.commit"
	ActionOrderAbate           = "order.abate"
	ActionChargeCreate         = "charge.create"
	ActionChargePay            = "charge.pay"
	ActionChargeRefund         = "charge.refund"
	ActionDispenseCreate       = "dispense.create"
	ActionDispenseReturn       = "dispense.return"
	ActionDepartmentCreate     = "department.create"
	ActionDepartmentUpdate     = "department.update"
	ActionDepartmentDeactivate = "department.deactivate"
	ActionDoctorCreate         = "doctor.create"
	ActionDoctorUpdate         = "doctor.update"
	ActionCatalogCreate        = "catalog.create"
	ActionCatalogUpdate        = "catalog.update"
	ActionInventoryAdjust      = "inventory.adjust"
	ActionDictCreate           = "dict.create"
	ActionDictUpdate           = "dict.update"
	ActionUserCreate           = "user.create"
	ActionUserUpdate           = "user.update"
	ActionUserDeactivate       = "user.deactivate"
	ActionUserResetPassword    = "user.reset_password"

	OutcomeSuccess = "SUCCESS"
	OutcomeFail    = "FAIL"
)

// AuditEvent is an append-only identity audit row (LOGIN_SUCCESS / LOGIN_FAILURE / LOGOUT).
type AuditEvent struct {
	ID            string
	OccurredAt    time.Time
	ActorID       string
	ActorUsername string
	ActorRoles    string
	ModuleID      string
	DepartmentID  string
	Action        string
	Description   string
	ResourceType  string
	ResourceID    string
	Outcome       string
	ErrorCode     string
	RequestID     string
	ClientIP      string
	ClientMAC     string
	BuildVersion  string
	DurationMS    int64
}

// UserOperationLog is the display-safe projection returned by audit queries.
// ID is only used to maintain a stable descending cursor and is never serialized.
type UserOperationLog struct {
	ID            string    `json:"-"`
	ModuleID      string    `json:"moduleId"`
	ModuleName    string    `json:"moduleName"`
	LogType       string    `json:"logType"`
	OccurredAt    time.Time `json:"occurredAt"`
	ActorUserID   string    `json:"actorUserId"`
	ActorUsername string    `json:"actorUsername"`
	DepartmentID  string    `json:"departmentId"`
	Description   string    `json:"description"`
	ClientIP      string    `json:"clientIp"`
	ClientMAC     string    `json:"clientMac"`
	BuildVersion  string    `json:"buildVersion"`
	RequestID     string    `json:"requestId"`
	Outcome       string    `json:"outcome"`
	ErrorCode     string    `json:"errorCode"`
}

// AuditModuleID maps the current event vocabulary to stable access-module IDs.
func AuditModuleID(action string) string {
	switch action {
	case ActionPatientCreate, ActionRegisterCreate, ActionRegisterCheckIn, ActionRegisterComplete, ActionRegisterCancel,
		ActionDiagnoseCreate, ActionDiagnoseCommit, ActionDiagnoseAbate,
		ActionOrderCreate, ActionOrderCommit, ActionOrderAbate, ActionChargeCreate, ActionChargePay,
		ActionChargeRefund, ActionDispenseCreate, ActionDispenseReturn:
		return "module-outpatient"
	case ActionLoginSuccess, ActionLoginFailure, ActionLogout, ActionDepartmentCreate, ActionDepartmentUpdate,
		ActionDepartmentDeactivate, ActionDoctorCreate, ActionDoctorUpdate, ActionCatalogCreate, ActionCatalogUpdate,
		ActionInventoryAdjust, ActionDictCreate, ActionDictUpdate, ActionUserCreate, ActionUserUpdate,
		ActionUserDeactivate, ActionUserResetPassword, "user.change_password", "role.create", "role.update",
		"permission.create", "permission.update", "user.permission_override", "module.create", "module.update",
		"category.create", "category.update", "mode.create", "mode.update":
		return "module-master"
	default:
		return ""
	}
}

// AuditDescription translates known event action codes into the display label.
func AuditDescription(action string) string {
	labels := map[string]string{
		ActionLoginSuccess: "用户登录", ActionLoginFailure: "用户登录失败", ActionLogout: "用户退出",
		ActionPatientCreate: "患者建档", ActionRegisterCreate: "挂号", ActionRegisterCheckIn: "签到", ActionRegisterComplete: "完成就诊",
		ActionRegisterCancel: "退号", ActionDiagnoseCreate: "保存诊断", ActionDiagnoseCommit: "提交诊断", ActionDiagnoseAbate: "作废诊断",
		ActionOrderCreate: "新建医嘱", ActionOrderCommit: "提交医嘱",
		ActionOrderAbate: "作废医嘱", ActionChargeCreate: "创建收费单", ActionChargePay: "收费",
		ActionChargeRefund: "退费", ActionDispenseCreate: "发药", ActionDispenseReturn: "退药",
		ActionDepartmentCreate: "创建科室", ActionDepartmentUpdate: "修改科室", ActionDepartmentDeactivate: "停用科室",
		ActionDoctorCreate: "创建医生", ActionDoctorUpdate: "修改医生", ActionCatalogCreate: "创建目录项",
		ActionCatalogUpdate: "修改目录项", ActionInventoryAdjust: "调整库存", ActionDictCreate: "创建字典项",
		ActionDictUpdate: "修改字典项", ActionUserCreate: "创建用户", ActionUserUpdate: "修改用户",
		ActionUserDeactivate: "停用用户", ActionUserResetPassword: "重置用户密码", "user.change_password": "修改密码",
		"role.create": "创建角色", "role.update": "修改角色", "permission.create": "创建权限",
		"permission.update": "修改权限", "user.permission_override": "修改用户权限例外",
		"module.create": "创建功能模块", "module.update": "修改功能模块", "category.create": "创建权限分类",
		"category.update": "修改权限分类", "mode.create": "创建授权方式", "mode.update": "修改授权方式",
	}
	if label, ok := labels[action]; ok {
		return label
	}
	return action
}
