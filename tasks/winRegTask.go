package tasks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"

	tacoexec "github.com/cloudradar-monitoring/tacoscript/exec"
	"github.com/cloudradar-monitoring/tacoscript/winreg"

	"github.com/cloudradar-monitoring/tacoscript/utils"

	"github.com/sirupsen/logrus"
)

type WinRegActionType int

type WinRegTaskBuilder struct {
}

const (
	ActionWinRegPresent WinRegActionType = iota + 1
	ActionWinRegAbsent
	ActionWinRegAbsentKey
)

var ErrUnknownWinRegAction = errors.New("unknown action")

func (wrtb WinRegTaskBuilder) Build(typeName, path string, params interface{}) (Task, error) {
	task := &WinRegTask{
		TypeName: typeName,
		Path:     path,
	}

	switch typeName {
	case WinRegPresent:
		task.ActionType = ActionWinRegPresent
	case WinRegAbsent:
		task.ActionType = ActionWinRegAbsent
	case WinRegAbsentKey:
		task.ActionType = ActionWinRegAbsentKey
	}

	errs := Build(typeName, path, params, task, nil)

	return task, errs.ToError()
}

type WinRegTask struct {
	ActionType WinRegActionType
	TypeName   string
	Path       string

	Name    string `taco:"name"`
	RegPath string `taco:"reg_path"`
	Val     string `taco:"value"`
	ValType string `taco:"type"`

	Require []string `taco:"require"`
	Creates []string `taco:"creates"`
	OnlyIf  []string `taco:"onlyif"`
	Unless  []string `taco:"unless"`

	Shell string `taco:"shell"`

	tracker *FieldStatusTracker

	Updated bool
}

func (wrt *WinRegTask) GetTypeName() string {
	return wrt.TypeName
}

func (wrt *WinRegTask) GetRequirements() []string {
	return wrt.Require
}

func (wrt *WinRegTask) Validate(goos string) error {
	errs := &utils.Errors{}

	if wrt.ActionType == 0 {
		errs.Add(fmt.Errorf("unknown win_reg task type: %s", wrt.TypeName))
		return errs.ToError()
	}

	err := ValidateRequired(wrt.RegPath, wrt.Path+"."+RegPathField)
	if err != nil {
		errs.Add(err)
		return errs.ToError()
	}

	err = winreg.HasValidRootKey(wrt.RegPath)
	if err != nil {
		errs.Add(err)
	}

	if wrt.ActionType == ActionWinRegPresent || wrt.ActionType == ActionWinRegAbsent {
		err = ValidateRequired(wrt.Name, wrt.Path+"."+NameField)
		if err != nil {
			errs.Add(err)
		}
	}

	if wrt.ActionType == ActionWinRegPresent {
		err = ValidateRequired(wrt.Val, wrt.Path+"."+ValField)
		if err != nil {
			errs.Add(err)
		}

		err = ValidateRequired(wrt.ValType, wrt.Path+"."+ValTypeField)
		if err != nil {
			errs.Add(err)
		}
	}

	return errs.ToError()
}

func (wrt *WinRegTask) GetPath() string {
	return wrt.Path
}

func (wrt *WinRegTask) String() string {
	return fmt.Sprintf("task '%s' at path '%s'", wrt.TypeName, wrt.GetPath())
}

func (wrt *WinRegTask) GetOnlyIfCmds() []string {
	return wrt.OnlyIf
}

func (wrt *WinRegTask) GetUnlessCmds() []string {
	return wrt.Unless
}

func (wrt *WinRegTask) GetCreatesFilesList() []string {
	return wrt.Creates
}

func (wrt *WinRegTask) GetTracker() (tracker *FieldStatusTracker) {
	if wrt.tracker == nil {
		wrt.tracker = newFieldStatusTracker()
	}
	return wrt.tracker
}

func (wrt *WinRegTask) IsChangeField(inputKey string) (excluded bool) {
	return false
}

type WinRegTaskExecutor struct {
	Runner    tacoexec.Runner
	FsManager *utils.FsManager
}

func (wrte *WinRegTaskExecutor) Execute(ctx context.Context, task Task) ExecutionResult {
	execRes := ExecutionResult{
		Name:    task.GetTypeName(),
		Comment: "registry not updated",
		Changes: make(map[string]string),
	}

	if runtime.GOOS != "windows" {
		execRes.Err = errors.New("win_reg tasks only supported on Microsoft Windows")
		return execRes
	}

	logrus.Debugf("will trigger '%s' task", task.GetPath())

	wrt, ok := task.(*WinRegTask)
	if !ok {
		execRes.Err = fmt.Errorf("cannot convert task '%v' to WinRegTask", task)
		return execRes
	}

	execRes.Name = wrt.GetTypeName()

	var stdoutBuf, stderrBuf bytes.Buffer
	execCtx := &tacoexec.Context{
		Ctx:          ctx,
		Path:         wrt.Path,
		StdoutWriter: &stdoutBuf,
		StderrWriter: &stderrBuf,
		Shell:        wrt.Shell,
	}

	logrus.Debugf("will check if the task '%s' should be executed", task.GetPath())
	skipReason, err := checkConditionals(execCtx, wrte.FsManager, wrte.Runner, wrt)
	if err != nil {
		execRes.Err = err
		return execRes
	}

	if skipReason != "" {
		logrus.Debugf("the task '%s' will be be skipped", execRes.Name)
		execRes.IsSkipped = true
		execRes.SkipReason = skipReason
		return execRes
	}

	start := time.Now()

	err = wrte.ExecuteTask(ctx, wrt, &execRes)
	if err != nil {
		execRes.Err = err
		return execRes
	}

	execRes.Duration = time.Since(start)

	logrus.Debugf("the task '%s' is finished for %v", execRes.Name, execRes.Duration)

	return execRes
}

func (wrte *WinRegTaskExecutor) ExecuteTask(ctx context.Context, t *WinRegTask, res *ExecutionResult) (err error) {
	var updated bool
	var desc string

	switch t.ActionType {
	case ActionWinRegPresent:
		updated, desc, err = winreg.SetValue(t.RegPath, t.Name, t.Val, winreg.REG_SZ)
		if err != nil {
			res.Err = err
			return err
		}
		t.Updated = updated
	case ActionWinRegAbsent:
		updated, desc, err = winreg.RemoveValue(t.RegPath, t.Name)
		if err != nil {
			res.Err = err
			return err
		}
		t.Updated = updated
	case ActionWinRegAbsentKey:
		updated, desc, err = winreg.RemoveKey(t.RegPath)
		if err != nil {
			res.Err = err
			return err
		}
		t.Updated = updated
	default:
		res.Err = ErrUnknownWinRegAction
		return err
	}

	if t.Updated {
		res.Comment = "registry updated"
		res.Changes["registry"] = desc
	}

	return nil
}