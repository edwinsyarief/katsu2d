
# State Machine

## Explanation of Key Components

- **Type Constraints:** The state (S) and event (E) types must be comparable because they are used as map keys. This ensures the state machine works with types like strings, integers, or custom enums.

- **Guard Functions:** The GuardFunc type allows transitions to be conditional. If guard is nil, the transition occurs unconditionally; otherwise, the guard must return true.

- **Flyweight Pattern:** The StateMachineConfig struct contains all shared data (transitions, enter/exit functions), while StateMachine holds only a pointer to the config and the current state. Multiple instances can share the same config, reducing memory usage.

- **Error Handling:** The Trigger method checks for valid transitions and guard conditions, returning descriptive errors if either fails.

## Usage Example

Here’s an example of using this state machine to model a simple enemy AI in a game:

```go
package main

import (
    "fmt"
    "statemachine"
)

type EnemyState string
type EnemyEvent string

const (
    Idle    EnemyState = "Idle"
    Patrol  EnemyState = "Patrol"
    Chase   EnemyState = "Chase"
    Attack  EnemyState = "Attack"
)

const (
    DetectPlayer EnemyEvent = "DetectPlayer"
    LosePlayer   EnemyEvent = "LosePlayer"
    InRange      EnemyEvent = "InRange"
    OutOfRange   EnemyEvent = "OutOfRange"
)

func main() {
    // Create shared configuration
    config := statemachine.NewStateMachineConfig[EnemyState, EnemyEvent](Idle)

    // Define transitions with guards
    playerInRange := func(state EnemyState, event EnemyEvent) bool {
        return true // Simulate player proximity; in a real game, check distance
    }

    config.AddTransition(Idle, DetectPlayer, Patrol, nil)
    config.AddTransition(Patrol, DetectPlayer, Chase, nil)
    config.AddTransition(Chase, InRange, Attack, playerInRange)
    config.AddTransition(Chase, LosePlayer, Patrol, nil)
    config.AddTransition(Attack, OutOfRange, Chase, nil)

    // Define enter/exit actions
    config.OnEnter(Attack, func(state EnemyState) {
        fmt.Printf("Entering %v: Enemy starts attacking!\n", state)
    })
    config.OnExit(Chase, func(state EnemyState) {
        fmt.Printf("Exiting %v: Enemy stops chasing.\n", state)
    })

    // Create two enemies sharing the same config
    enemy1 := statemachine.NewStateMachine(config)
    enemy2 := statemachine.NewStateMachine(config)

    // Simulate enemy1 behavior
    fmt.Println("Enemy1 state:", enemy1.CurrentState())
    if err := enemy1.Trigger(DetectPlayer); err != nil {
        fmt.Println("Error:", err)
    }
    fmt.Println("Enemy1 state:", enemy1.CurrentState())
    if err := enemy1.Trigger(DetectPlayer); err != nil {
        fmt.Println("Error:", err)
    }
    fmt.Println("Enemy1 state:", enemy1.CurrentState())
    if err := enemy1.Trigger(InRange); err != nil {
        fmt.Println("Error:", err)
    }
    fmt.Println("Enemy1 state:", enemy1.CurrentState())

    // Simulate enemy2 behavior
    fmt.Println("Enemy2 state:", enemy2.CurrentState())
    if err := enemy2.Trigger(DetectPlayer); err != nil {
        fmt.Println("Error:", err)
    }
    fmt.Println("Enemy2 state:", enemy2.CurrentState())
}
```
