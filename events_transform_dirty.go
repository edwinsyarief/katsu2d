package katsu2d

import "github.com/edwinsyarief/lazyecs"

// Define event
type TransformZDirtyEvent struct{ Entity lazyecs.Entity }
