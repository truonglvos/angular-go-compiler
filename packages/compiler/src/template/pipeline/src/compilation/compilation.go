package compilation

import (
	"ngc-go/packages/compiler/src/output"
	constant_pool "ngc-go/packages/compiler/src/pool"
	"ngc-go/packages/compiler/src/render3/view"
	ir "ngc-go/packages/compiler/src/template/pipeline/ir/src"
	ir_operations "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ir_variables "ngc-go/packages/compiler/src/template/pipeline/ir/src/variable"
)

// CompilationJobKind represents the kind of compilation job
type CompilationJobKind int

const (
	// CompilationJobKindTmpl - Template compilation
	CompilationJobKindTmpl CompilationJobKind = iota
	// CompilationJobKindHost - Host binding compilation
	CompilationJobKindHost
	// CompilationJobKindBoth - A special value used to indicate that some logic applies to both compilation types
	CompilationJobKindBoth
)

// TemplateCompilationMode represents possible modes in which a component's template can be compiled
type TemplateCompilationMode int

const (
	// TemplateCompilationModeFull - Supports the full instruction set, including directives
	TemplateCompilationModeFull TemplateCompilationMode = iota
	// TemplateCompilationModeDomOnly - Uses a narrower instruction set that doesn't support directives and allows optimizations
	TemplateCompilationModeDomOnly
)

// CompilationJob is an entire ongoing compilation, which will result in one or more template functions when complete.
// Contains one or more corresponding compilation units.
type CompilationJob struct {
	ComponentName string
	Pool          *constant_pool.ConstantPool
	Compatibility ir.CompatibilityMode
	Mode          TemplateCompilationMode
	Kind          CompilationJobKind
	nextXrefId    ir_operations.XrefId
}

// NewCompilationJob creates a new CompilationJob
func NewCompilationJob(
	componentName string,
	pool *constant_pool.ConstantPool,
	compatibility ir.CompatibilityMode,
	mode TemplateCompilationMode,
) *CompilationJob {
	return &CompilationJob{
		ComponentName: componentName,
		Pool:          pool,
		Compatibility: compatibility,
		Mode:          mode,
		Kind:          CompilationJobKindBoth,
		nextXrefId:    0,
	}
}

// AllocateXrefId generates a new unique `ir.XrefId` in this job
func (j *CompilationJob) AllocateXrefId() ir_operations.XrefId {
	id := j.nextXrefId
	j.nextXrefId++
	return id
}

// GetUnits returns all compilation units in this job
func (j *CompilationJob) GetUnits() []CompilationUnit {
	// This will be implemented by concrete types
	return nil
}

// GetRoot returns the root compilation unit
func (j *CompilationJob) GetRoot() CompilationUnit {
	// This will be implemented by concrete types
	return nil
}

// GetFnSuffix returns a unique string used to identify this kind of job
func (j *CompilationJob) GetFnSuffix() string {
	// This will be implemented by concrete types
	return ""
}

// ComponentCompilationJob is compilation-in-progress of a whole component's template,
// including the main template and any embedded views or host bindings.
type ComponentCompilationJob struct {
	*CompilationJob
	Root                    *ViewCompilationUnit
	Views                   map[ir_operations.XrefId]*ViewCompilationUnit
	ContentSelectors        output.OutputExpression
	Consts                  []output.OutputExpression
	ConstsInitializers      []output.OutputStatement
	RelativeContextFilePath string
	I18nUseExternalIds      bool
	DeferMeta               view.R3ComponentDeferMetadata
	AllDeferrableDepsFn     *output.ReadVarExpr
	RelativeTemplatePath    *string
	EnableDebugLocations    bool
}

// NewComponentCompilationJob creates a new ComponentCompilationJob
func NewComponentCompilationJob(
	componentName string,
	pool *constant_pool.ConstantPool,
	compatibility ir.CompatibilityMode,
	mode TemplateCompilationMode,
	relativeContextFilePath string,
	i18nUseExternalIds bool,
	deferMeta view.R3ComponentDeferMetadata,
	allDeferrableDepsFn *output.ReadVarExpr,
	relativeTemplatePath *string,
	enableDebugLocations bool,
) *ComponentCompilationJob {
	job := &ComponentCompilationJob{
		CompilationJob:          NewCompilationJob(componentName, pool, compatibility, mode),
		Views:                   make(map[ir_operations.XrefId]*ViewCompilationUnit),
		RelativeContextFilePath: relativeContextFilePath,
		I18nUseExternalIds:      i18nUseExternalIds,
		DeferMeta:               deferMeta,
		AllDeferrableDepsFn:     allDeferrableDepsFn,
		RelativeTemplatePath:    relativeTemplatePath,
		EnableDebugLocations:    enableDebugLocations,
	}
	job.CompilationJob.Kind = CompilationJobKindTmpl
	root := NewViewCompilationUnit(job, job.AllocateXrefId(), nil)
	job.Root = root
	job.Views[root.Xref] = root
	return job
}

// AllocateView adds a `ViewCompilationUnit` for a new embedded view to this compilation
func (j *ComponentCompilationJob) AllocateView(parent ir_operations.XrefId) *ViewCompilationUnit {
	view := NewViewCompilationUnit(j, j.AllocateXrefId(), &parent)
	j.Views[view.Xref] = view
	return view
}

// GetUnits returns all view compilation units
func (j *ComponentCompilationJob) GetUnits() []CompilationUnit {
	units := make([]CompilationUnit, 0, len(j.Views))
	for _, view := range j.Views {
		units = append(units, view)
	}
	return units
}

// GetRoot returns the root view compilation unit
func (j *ComponentCompilationJob) GetRoot() CompilationUnit {
	return j.Root
}

// GetFnSuffix returns the function suffix for template compilation
func (j *ComponentCompilationJob) GetFnSuffix() string {
	return "Template"
}

// AddConst adds a constant `o.Expression` to the compilation and returns its index in the `consts` array
func (j *ComponentCompilationJob) AddConst(newConst output.OutputExpression, initializers []output.OutputStatement) ir_operations.ConstIndex {
	for idx := 0; idx < len(j.Consts); idx++ {
		if j.Consts[idx].IsEquivalent(newConst) {
			return ir_operations.ConstIndex(idx)
		}
	}
	idx := len(j.Consts)
	j.Consts = append(j.Consts, newConst)
	if initializers != nil {
		j.ConstsInitializers = append(j.ConstsInitializers, initializers...)
	}
	return ir_operations.ConstIndex(idx)
}

// CompilationUnit is compiled into a template function. Some example units are views and host bindings.
type CompilationUnit interface {
	GetXref() ir_operations.XrefId
	GetJob() *CompilationJob
	GetCreate() *ir_operations.OpList
	GetUpdate() *ir_operations.OpList
	GetFnName() *string
	SetFnName(name string)
	GetVars() *int
	SetVars(vars int)
}

// ViewCompilationUnit is compilation-in-progress of an individual view within a template.
type ViewCompilationUnit struct {
	Job              *ComponentCompilationJob
	Xref             ir_operations.XrefId
	Parent           *ir_operations.XrefId
	Create           *ir_operations.OpList
	Update           *ir_operations.OpList
	FnName           *string
	Vars             *int
	ContextVariables map[string]string
	Aliases          map[ir_variables.AliasVariable]bool
	Decls            *int
}

// NewViewCompilationUnit creates a new ViewCompilationUnit
func NewViewCompilationUnit(
	job *ComponentCompilationJob,
	xref ir_operations.XrefId,
	parent *ir_operations.XrefId,
) *ViewCompilationUnit {
	return &ViewCompilationUnit{
		Job:              job,
		Xref:             xref,
		Parent:           parent,
		Create:           ir_operations.NewOpList(),
		Update:           ir_operations.NewOpList(),
		ContextVariables: make(map[string]string),
		Aliases:          make(map[ir_variables.AliasVariable]bool),
	}
}

// GetXref returns the xref ID
func (v *ViewCompilationUnit) GetXref() ir_operations.XrefId {
	return v.Xref
}

// GetJob returns the compilation job
func (v *ViewCompilationUnit) GetJob() *CompilationJob {
	return v.Job.CompilationJob
}

// GetCreate returns the create operations list
func (v *ViewCompilationUnit) GetCreate() *ir_operations.OpList {
	return v.Create
}

// GetUpdate returns the update operations list
func (v *ViewCompilationUnit) GetUpdate() *ir_operations.OpList {
	return v.Update
}

// GetFnName returns the function name
func (v *ViewCompilationUnit) GetFnName() *string {
	return v.FnName
}

// SetFnName sets the function name
func (v *ViewCompilationUnit) SetFnName(name string) {
	v.FnName = &name
}

// GetVars returns the number of variable slots
func (v *ViewCompilationUnit) GetVars() *int {
	return v.Vars
}

// SetVars sets the number of variable slots
func (v *ViewCompilationUnit) SetVars(vars int) {
	v.Vars = &vars
}

// HostBindingCompilationJob is compilation-in-progress of a host binding,
// which contains a single unit for that host binding.
type HostBindingCompilationJob struct {
	*CompilationJob
	Root *HostBindingCompilationUnit
}

// NewHostBindingCompilationJob creates a new HostBindingCompilationJob
func NewHostBindingCompilationJob(
	componentName string,
	pool *constant_pool.ConstantPool,
	compatibility ir.CompatibilityMode,
	mode TemplateCompilationMode,
) *HostBindingCompilationJob {
	job := &HostBindingCompilationJob{
		CompilationJob: NewCompilationJob(componentName, pool, compatibility, mode),
	}
	job.CompilationJob.Kind = CompilationJobKindHost
	root := NewHostBindingCompilationUnit(job)
	job.Root = root
	return job
}

// GetUnits returns all host binding compilation units
func (j *HostBindingCompilationJob) GetUnits() []CompilationUnit {
	return []CompilationUnit{j.Root}
}

// GetRoot returns the root host binding compilation unit
func (j *HostBindingCompilationJob) GetRoot() CompilationUnit {
	return j.Root
}

// GetFnSuffix returns the function suffix for host binding compilation
func (j *HostBindingCompilationJob) GetFnSuffix() string {
	return "HostBindings"
}

// HostBindingCompilationUnit is a compilation unit for host bindings
type HostBindingCompilationUnit struct {
	Job        *HostBindingCompilationJob
	Xref       ir_operations.XrefId
	Create     *ir_operations.OpList
	Update     *ir_operations.OpList
	FnName     *string
	Vars       *int
	Attributes *output.LiteralArrayExpr
}

// NewHostBindingCompilationUnit creates a new HostBindingCompilationUnit
func NewHostBindingCompilationUnit(job *HostBindingCompilationJob) *HostBindingCompilationUnit {
	return &HostBindingCompilationUnit{
		Job:    job,
		Xref:   0,
		Create: ir_operations.NewOpList(),
		Update: ir_operations.NewOpList(),
	}
}

// GetXref returns the xref ID
func (h *HostBindingCompilationUnit) GetXref() ir_operations.XrefId {
	return h.Xref
}

// GetJob returns the compilation job
func (h *HostBindingCompilationUnit) GetJob() *CompilationJob {
	return h.Job.CompilationJob
}

// GetCreate returns the create operations list
func (h *HostBindingCompilationUnit) GetCreate() *ir_operations.OpList {
	return h.Create
}

// GetUpdate returns the update operations list
func (h *HostBindingCompilationUnit) GetUpdate() *ir_operations.OpList {
	return h.Update
}

// GetFnName returns the function name
func (h *HostBindingCompilationUnit) GetFnName() *string {
	return h.FnName
}

// SetFnName sets the function name
func (h *HostBindingCompilationUnit) SetFnName(name string) {
	h.FnName = &name
}

// GetVars returns the number of variable slots
func (h *HostBindingCompilationUnit) GetVars() *int {
	return h.Vars
}

// SetVars sets the number of variable slots
func (h *HostBindingCompilationUnit) SetVars(vars int) {
	h.Vars = &vars
}
