package fsm

import "fmt"

// EnterFunc defines a function called when entering a state.
type EnterFunc[S any] func(state S)

// ExitFunc defines a function called when exiting a state.
type ExitFunc[S any] func(state S)

// GuardFunc defines a guard condition for a transition.
type GuardFunc[S, E any] func(currentState S, event E) bool

// Transition holds the next state and an optional guard.
type Transition[S, E any] struct {
	nextState S
	guard     GuardFunc[S, E]
}

// StateEventKey is used as a key in the transitions map.
type StateEventKey[S, E any] struct {
	state S
	event E
}

// StateMachineConfig holds the shared configuration for state machines.
type StateMachineConfig[S comparable, E comparable] struct {
	initialState S
	transitions  map[StateEventKey[S, E]]Transition[S, E]
	enterFuncs   map[S]EnterFunc[S]
	exitFuncs    map[S]ExitFunc[S]
}

// NewStateMachineConfig creates a new configuration with the given initial state.
func NewStateMachineConfig[S comparable, E comparable](initialState S) *StateMachineConfig[S, E] {
	return &StateMachineConfig[S, E]{
		initialState: initialState,
		transitions:  make(map[StateEventKey[S, E]]Transition[S, E]),
		enterFuncs:   make(map[S]EnterFunc[S]),
		exitFuncs:    make(map[S]ExitFunc[S]),
	}
}

// AddTransition adds a transition with an optional guard.
func (c *StateMachineConfig[S, E]) AddTransition(from S, event E, to S, guard GuardFunc[S, E]) {
	key := StateEventKey[S, E]{state: from, event: event}
	c.transitions[key] = Transition[S, E]{nextState: to, guard: guard}
}

// OnEnter registers a function to be called when entering a state.
func (c *StateMachineConfig[S, E]) OnEnter(state S, f EnterFunc[S]) {
	c.enterFuncs[state] = f
}

// OnExit registers a function to be called when exiting a state.
func (c *StateMachineConfig[S, E]) OnExit(state S, f ExitFunc[S]) {
	c.exitFuncs[state] = f
}

// StateMachine is a finite state machine using the shared configuration.
type StateMachine[S comparable, E comparable] struct {
	config       *StateMachineConfig[S, E]
	currentState S
}

// NewStateMachine creates a new state machine with the given configuration.
func NewStateMachine[S comparable, E comparable](config *StateMachineConfig[S, E]) *StateMachine[S, E] {
	return &StateMachine[S, E]{
		config:       config,
		currentState: config.initialState,
	}
}

// Trigger processes an event, potentially transitioning to a new state.
func (sm *StateMachine[S, E]) Trigger(event E) error {
	key := StateEventKey[S, E]{state: sm.currentState, event: event}
	transition, ok := sm.config.transitions[key]
	if !ok {
		return fmt.Errorf("no transition for state %v and event %v", sm.currentState, event)
	}
	if transition.guard != nil && !transition.guard(sm.currentState, event) {
		return fmt.Errorf("guard condition not met for transition from %v on event %v", sm.currentState, event)
	}
	// Perform transition
	if exitFunc, ok := sm.config.exitFuncs[sm.currentState]; ok && exitFunc != nil {
		exitFunc(sm.currentState)
	}
	sm.currentState = transition.nextState
	if enterFunc, ok := sm.config.enterFuncs[sm.currentState]; ok && enterFunc != nil {
		enterFunc(sm.currentState)
	}
	return nil
}

// CurrentState returns the current state.
func (sm *StateMachine[S, E]) CurrentState() S {
	return sm.currentState
}

// Reset sets the state machine back to its initial state.
func (sm *StateMachine[S, E]) Reset() {
	sm.currentState = sm.config.initialState
}
