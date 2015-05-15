package httprest

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/netrack/netrack/httprest/format"
	"github.com/netrack/netrack/httprest/v1/models"
	"github.com/netrack/netrack/httputil"
	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
)

type MechanismContext struct {
	// Back-end context
	Mech *mech.MechanismContext

	// Mechanism manager instance.
	Manager mech.MechanismManager

	// Mechanism name
	Mechanism mech.Mechanism

	// Write formatter
	W format.WriteFormatter

	// Read formatter
	R format.ReadFormatter
}

func mechanismState(m mech.Mechanism) string {
	var repr []string

	if m.Enabled() {
		repr = append(repr, "enabled")
	}

	if m.Activated() {
		repr = append(repr, "activated")
	}

	if len(repr) == 0 {
		repr = append(repr, "disabled")
	}

	return strings.Join(repr, ",")
}

type MechanismHandler struct {
	c  *mech.HTTPDriverContext
	fn func(func(interface{}) error) (mech.MechanismManager, error)
}

func (h *MechanismHandler) context(rw http.ResponseWriter, r *http.Request) (*MechanismContext, error) {
	log.InfoLog("mechanism_handlers/CONTEXT",
		"Got request to handle mechanisms")

	dpid := httputil.Param(r, "dpid")
	mname := httputil.Param(r, "mechanism")

	rf, wf := Format(r)

	log.DebugLogf("mechanism_handlers/CONTEXT",
		"Request to handle mechanism %s of %s", mname, dpid)

	context, err := h.c.SwitchManager.Context(dpid)
	if err != nil {
		log.ErrorLog("mechanism_handlers/CONTEXT",
			"Failed to find requested datapath: ", err)

		text := fmt.Sprintf("switch '%s' not found", dpid)

		wf.Write(rw, models.Error{text}, http.StatusNotFound)
		return nil, fmt.Errorf(text)
	}

	var manager mech.MechanismManager
	if manager, err = h.fn(context.Managers.Obtain); err != nil {
		log.ErrorLog("mechanism_handlers/MECHANISM_MANAGER",
			"Failed to obtain mechanism manager: ", err)

		text := fmt.Sprintf("mechanism manager is dead")
		wf.Write(rw, models.Error{text}, http.StatusInternalServerError)
		return nil, err
	}

	var mechanism mech.Mechanism
	if mname != "" {
		mechanism, err = manager.Mechanism(mname)
		if err != nil {
			log.ErrorLog("mechanism_handlers/MECHANISM_MANAGER",
				"Failed to find requested mechanism: ", err)

			text := fmt.Sprintf("mechanism '%s' not registered", mname)
			wf.Write(rw, models.Error{text}, http.StatusNotFound)
			return nil, err
		}
	}

	ctx := &MechanismContext{
		Mechanism: mechanism,
		Manager:   manager,
		Mech:      context,
		W:         wf,
		R:         rf,
	}

	return ctx, nil
}

func (h *MechanismHandler) indexHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("mechanism_handlers/INDEX_HANDLER",
		"Got request to list mechanisms")

	context, err := h.context(rw, r)
	if err != nil {
		return
	}

	mechanismModels := make([]models.Mechanism, 0)
	for _, mechanism := range context.Manager.MechanismList() {
		mechanismModels = append(mechanismModels, models.Mechanism{
			Name:        mechanism.Name(),
			Description: mechanism.Description(),
			State:       mechanismState(mechanism),
		})
	}

	context.W.Write(rw, mechanismModels, http.StatusOK)
}

func (h *MechanismHandler) showHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("mechanism_handlers/SHOW_HANDLER",
		"Got request to show mechanism")

	context, err := h.context(rw, r)
	if err != nil {
		return
	}

	mechanismModel := models.Mechanism{
		Name:        context.Mechanism.Name(),
		Description: context.Mechanism.Description(),
		State:       mechanismState(context.Mechanism),
	}

	context.W.Write(rw, mechanismModel, http.StatusOK)
}

func (h *MechanismHandler) enableHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("mechanism_handlers/ENABLE_HANDLER",
		"Got request to enable mechanism")

	context, err := h.context(rw, r)
	if err != nil {
		return
	}

	err = context.Manager.EnableByName(context.Mechanism.Name(), context.Mech)
	if err != nil {
		log.ErrorLog("mechanism_handlers/ENABLE_HANDLER",
			"Failed to enable requested mechanism: ", err)

		text := fmt.Sprintf("failed to enable '%s' mechanism", context.Mechanism.Name())
		context.W.Write(rw, models.Error{text}, http.StatusConflict)
		return
	}

	err = context.Manager.ActivateByName(context.Mechanism.Name())
	if err != nil {
		log.ErrorLog("mechanism_handlers/ENABLE_HADNLER",
			"Failed to activate requested mechanism: ", err)

		text := fmt.Sprintf("failed to activate '%s' mechanism", context.Mechanism.Name())
		context.W.Write(rw, models.Error{text}, http.StatusConflict)
		return
	}

	context.W.Write(rw, nil, http.StatusOK)
}

func (h *MechanismHandler) disableHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("mechanism_handlers/DISABLE_HANDLER",
		"Got request to disable mechanism")

	context, err := h.context(rw, r)
	if err != nil {
		return
	}

	err = context.Manager.DisableByName(context.Mechanism.Name())
	if err != nil {
		log.ErrorLog("mechanism_handlers/DISABLE_HANDLER",
			"Failed to disable requested mechanism: ", err)

		text := fmt.Sprintf("failed to disable '%s' mechanism", context.Mechanism.Name())
		context.W.Write(rw, models.Error{text}, http.StatusConflict)
		return
	}

	context.W.Write(rw, nil, http.StatusOK)
}
