package models

type SheetLastInfo struct {
	Date  string `json:"last_updated_date,omitempty"`
	Task  string `json:"last_task_info,omitempty"`
	Hours string `json:"last_working_hours,omitempty"`
	Leave string `json:"leave_on_last_date,omitempty"`
}

type SheetNewinfo struct {
	Date  string `json:"date_to_update,omitempty"`
	Task  string `json:"ltask_done,omitempty"`
	Hours string `json:"total_hours,omitempty"`
	Leave string `json:"on_leave,omitempty"`
}

type Timesheet struct {
	Lastupdate *SheetLastInfo `json:"last_update,omitempty"`
	NewEntry   *SheetNewinfo  `json:"upcoming_change,omitempty"`
}
