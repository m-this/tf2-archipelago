package form

import (
	"errors"
	"fmt"
	"slices"
)

/*
Build resolves the specs against one state into a screen.

Pure, and that is the point of it: the same state and the same Env give the same
Model, so a test asserts on what the player would be shown without opening a
window or a terminal. Everything a Spec left as a closure is a value by the time
it comes back.

An empty page is dropped rather than drawn. A page with a title and no rows is
one the player opens once and never again, and the Missions page really can be
empty: no community pack on disk and every Valve mission excluded leaves it with
its buttons and nothing to tick.
*/
func Build(s State, env Env) Model {
	specs := Specs(s, env)
	byTab := make(map[string][]Field, len(Tabs))
	for _, spec := range specs {
		byTab[spec.Tab] = append(byTab[spec.Tab], spec.field(s, env))
	}

	model := Model{}
	for _, title := range Tabs {
		fields := byTab[title]
		if len(fields) == 0 {
			continue
		}
		model.Tabs = append(model.Tabs, Tab{
			Title:  title,
			Intro:  Intros[title],
			Under:  Nested[title],
			Fields: fields,
		})
	}
	return model
}

// Page is one page of the model by title, for an interface that lays its pages
// out itself. A page the specs no longer declare comes back empty rather than
// missing, so a window drawing it gets a blank tab instead of a panic.
func Page(model Model, title string) (Tab, bool) {
	for _, tab := range model.Tabs {
		if tab.Title == title {
			return tab, true
		}
	}
	return Tab{Title: title}, false
}

// field is one spec resolved. Kept beside Build rather than on the interfaces,
// so there is one answer to what a row looks like.
func (spec Spec) field(s State, env Env) Field {
	f := Field{
		ID:          spec.ID,
		Kind:        spec.Kind,
		Label:       spec.Label,
		Help:        spec.Help,
		Group:       spec.Group,
		Bar:         spec.Bar,
		Placeholder: spec.Placeholder,
		Hint:        spec.Hint,
		HintOff:     spec.HintOff,
		Warning:     spec.Warning,
		Browse:      spec.Browse,
		Deferred:    spec.Deferred,
	}
	if spec.Get != nil {
		f.Value = spec.Get(s)
	}
	if spec.Bounds != nil {
		f.Low, f.High = spec.Bounds(s, env)
	}
	if spec.Options != nil {
		f.Options = spec.Options(s, env)
	}
	if spec.Unavailable != nil {
		if reason := spec.Unavailable(s, env); reason != "" {
			f.Disabled, f.Reason = true, reason
		}
	}
	return f
}

/*
Apply writes one answer back and returns the state that results.

It takes and returns a State by value, so a refused change leaves the caller's
copy alone and there is no half-written state to undo. The caller rebuilds the
Model from what comes back; nothing patches a Field in place, because a change
to one row can move another row's bounds and can add and remove rows entirely.
Ticking a community pack is the clearest case: the Missions page grows a row per
mission of that pack.

An Action has no value to apply. Apply says so rather than ignoring it, because
an interface that routes a button press here instead of to its dispatcher has a
bug, and a silent no-op is how it survives to the next release.
*/
func Apply(s State, env Env, c Change) (State, error) {
	spec, ok := specByID(s, env, c.Field)
	if !ok {
		return s, fmt.Errorf("no setting %q", c.Field)
	}
	if spec.Set == nil {
		return s, fmt.Errorf("%q is a %s and is dispatched, not applied", c.Field, spec.Kind)
	}
	if spec.Unavailable != nil {
		if reason := spec.Unavailable(s, env); reason != "" {
			return s, errors.New(reason)
		}
	}
	return spec.Set(s, c.Value)
}

// Dispatchable reports whether the ID names an Action or a Confirm, which is
// what an interface asks before sending a press anywhere.
func Dispatchable(s State, env Env, id string) bool {
	spec, ok := specByID(s, env, id)
	return ok && (spec.Kind == Action || spec.Kind == Confirm)
}

func specByID(s State, env Env, id string) (Spec, bool) {
	specs := Specs(s, env)
	i := slices.IndexFunc(specs, func(spec Spec) bool { return spec.ID == id })
	if i < 0 {
		return Spec{}, false
	}
	return specs[i], true
}

// String names a Kind for an error message. Not for the player: an interface
// picks its own word for a row it is drawing.
func (k Kind) String() string {
	switch k {
	case Text:
		return "text"
	case Password:
		return "password"
	case Number:
		return "number"
	case Toggle:
		return "toggle"
	case Choice:
		return "choice"
	case Action:
		return "action"
	case Confirm:
		return "confirm"
	}
	return fmt.Sprintf("kind(%d)", uint8(k))
}
