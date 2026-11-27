package katsu2d

// Transformer defines the interface for objects that have transforms.
type Transformer interface {
	GetParentTransform() *Transform
	GetTransform() *Transform
}

// Transform represents a transformation in 2D space.
// It includes position, scale, rotation, and a parent for hierarchical transforms.
type Transform struct {
	parent                          *Transform
	worldMatrix                     Matrix
	parentMatrix                    Matrix
	parentInverted                  Matrix
	position, scale, offset, origin Vector
	rotation                        float64
	isDirty                         bool // A single dirty flag for performance caching.
}

// T creates a new Transform with default values.
// The default scale is 1, and the dirty flag is set to true.
func T() *Transform {
	return &Transform{
		scale:   V2(1),
		isDirty: true,
	}
}

// Reset sets the transform's local properties to their default (identity) values.
// Position is set to (0, 0), Scale to (1, 1), Rotation to 0, and Offset/Origin to (0, 0).
// The parent relationship is preserved.
func (t *Transform) Reset() {
	t.position = V2(0) // Default position (0, 0)
	t.scale = V2(1)    // Default scale (1, 1)
	t.offset = V2(0)   // Default offset (0, 0)
	t.origin = V2(0)   // Default origin (0, 0)
	t.rotation = 0.0   // Default rotation (0.0)
	// Mark as dirty to force world matrix recalculation next time Matrix() is called.
	t.isDirty = true
}

// Methods for Parent Hierarchy
// ----------------------------

// GetParentTransform returns the parent Transform or nil if there is no parent.
func (t *Transform) GetParentTransform() *Transform {
	return t.parent
}

// GetInitialParentTransform finds the topmost parent in the hierarchy.
// It iterates up the parent chain until it finds the root transform.
func (t *Transform) GetInitialParentTransform() *Transform {
	curr := t
	for curr.parent != nil {
		curr = curr.parent
	}
	return curr
}

// GetTransform returns this Transform.
// This method fulfills the Transformer interface.
func (t *Transform) GetTransform() *Transform {
	return t
}

// Transformation Properties
// -------------------------

// Origin returns the origin of the transform.
func (t *Transform) Origin() Vector {
	return t.origin
}

// SetOrigin updates the transform's origin and marks it as dirty.
func (t *Transform) SetOrigin(origin Vector) {
	t.isDirty = true
	t.origin = origin
}

// IsDirty checks if this transform or any of its parents are dirty.
// This recursive check is used for cache invalidation.
func (t *Transform) IsDirty() bool {
	// A single, recursive check for dirty state.
	if t.isDirty {
		return true
	}
	if t.parent != nil {
		return t.parent.IsDirty()
	}
	return false
}

// Position and Movement
// ---------------------

// SetPosition updates the position, preserving the world-space position
// by adjusting the local position based on the parent's inverse matrix.
func (t *Transform) SetPosition(position Vector) {
	t.isDirty = true
	if t.parent != nil {
		// Calculate the local position by transforming the world position by the parent's inverse matrix.
		worldToLocalMatrix := t.parent.Matrix()
		worldToLocalMatrix.Invert()
		t.position = position.Apply(worldToLocalMatrix)
	} else {
		// If there is no parent, the local position is the world position.
		t.position = position
	}
}

// Position returns the local position of the transform.
func (t *Transform) Position() Vector {
	return t.position
}

// Move translates the transform by the given vector(s).
func (t *Transform) Move(v ...Vector) {
	t.SetPosition(t.Position().Add(v...))
}

// Rotation
// --------

// SetRotation updates the rotation, preserving the world-space rotation
// by adjusting the local rotation based on the parent's rotation.
func (t *Transform) SetRotation(rotation float64) {
	t.isDirty = true
	if t.parent != nil {
		// Calculate the local rotation by subtracting the parent's world rotation.
		t.rotation = rotation - t.parent.Rotation()
	} else {
		// If there is no parent, the local rotation is the world rotation.
		t.rotation = rotation
	}
}

// Rotation returns the absolute rotation in world space.
func (t *Transform) Rotation() float64 {
	if t.parent == nil {
		return t.rotation
	}
	return t.rotation + t.parent.Rotation()
}

// Rotate adds to the current rotation.
func (t *Transform) Rotate(rotation float64) {
	t.isDirty = true
	t.rotation += rotation
}

// Scale
// -----

// SetScale updates the local scale.
func (t *Transform) SetScale(scale Vector) {
	t.isDirty = true
	t.scale = scale
}

// Scale returns the absolute scale in world space by multiplying with the parent's scale.
func (t *Transform) Scale() Vector {
	if t.parent == nil {
		return t.scale
	}
	return t.scale.Scale(t.parent.Scale())
}

// AddScale adds to the current local scale.
func (t *Transform) AddScale(add ...Vector) {
	t.isDirty = true
	t.scale = t.scale.Add(add...)
}

// Offset
// ------

// SetOffset updates the local offset.
func (t *Transform) SetOffset(offset Vector) {
	t.isDirty = true
	t.offset = offset
}

// Offset returns the local offset.
func (t *Transform) Offset() Vector {
	return t.offset
}

// Transform Modifiers
// -------------------

// Abs returns a new transform with the absolute world properties of this transform,
// effectively disconnecting it from its parent hierarchy.
func (t *Transform) Abs() Transform {
	abs := *T()
	abs.SetPosition(t.Position())
	abs.SetRotation(t.Rotation())
	abs.SetScale(t.Scale())
	abs.SetOffset(t.Offset())
	abs.SetOrigin(t.Origin())
	return abs
}

// Rel returns a copy of the transform with its parent set to nil.
func (t *Transform) Rel() Transform {
	rel := *t
	rel.parent = nil
	return rel
}

// Parent Management
// -----------------

// Connected returns true if the transform has a parent.
func (t *Transform) Connected() bool {
	return t.parent != nil
}

// Replace updates this transform's local properties to match the world properties
// of another transform.
func (t *Transform) Replace(new Transformer) {
	nt := new.GetTransform()
	t.SetPosition(nt.Position())
	t.SetOffset(nt.Offset())
	t.SetRotation(nt.Rotation())
	t.SetOrigin(nt.Origin())
	t.SetScale(nt.Scale())
}

// Connect establishes a parent-child relationship, preserving the object's
// world space transform.
func (t *Transform) Connect(parent Transformer) {
	if parent == nil {
		return
	}
	// Store the current world properties before connecting to the parent.
	worldPos := t.Position()
	worldRot := t.Rotation()
	worldScale := t.Scale()
	worldOffset := t.Offset()
	worldOrigin := t.Origin()
	// Set the new parent and mark the transform as dirty.
	t.parent = parent.GetTransform()
	t.isDirty = true
	// Re-apply the stored world properties, which will internally
	// calculate the new local properties based on the new parent.
	t.SetPosition(worldPos)
	t.SetRotation(worldRot)
	t.SetScale(worldScale)
	t.SetOffset(worldOffset)
	t.SetOrigin(worldOrigin)
}

// Disconnect removes the parent relationship, making the transform absolute.
// It does this by creating a new absolute transform and overwriting the current one.
func (t *Transform) Disconnect() {
	if t.parent == nil {
		return
	}
	*t = t.Abs()
}

// Matrix Operations
// -----------------

// MatrixForParenting returns matrices for child positioning.
// It returns the world matrix without the origin offset and its inverse.
// It ensures the world matrix is up-to-date by calling Matrix() if needed.
func (t *Transform) MatrixForParenting() (Matrix, Matrix) {
	if t.isDirty {
		t.Matrix() // Ensure the world matrix is computed and cached.
	}
	return t.parentMatrix, t.parentInverted
}

// Matrix computes the full world transformation matrix for this node.
// The result is cached to avoid repeated calculations.
func (t *Transform) Matrix() Matrix {
	if !t.IsDirty() {
		return t.worldMatrix
	}
	// Calculate the local transformation matrix.
	localMatrix := Matrix{} // Scale first.
	localMatrix.Scale(t.scale.X, t.scale.Y)
	// Then move and rotate.
	localMatrix.Translate(
		-t.offset.X*t.scale.X,
		-t.offset.Y*t.scale.Y,
	)
	localMatrix.Rotate(t.rotation)
	// Move to the absolute position.
	localMatrix.Translate(t.position.X, t.position.Y)
	// Save this local matrix as the parent matrix for children, without the origin offset.
	t.parentMatrix = localMatrix
	t.parentInverted = localMatrix
	t.parentInverted.Invert()
	// And finally move to the origin offset
	if !t.origin.IsZero() {
		localMatrix.Translate(-t.origin.X, -t.origin.Y)
	}
	// If there's a parent, combine this local matrix with the parent's world matrix.
	if t.parent != nil {
		parentMatrix := t.parent.Matrix()
		localMatrix.Concat(parentMatrix)
	}
	t.worldMatrix = localMatrix
	t.isDirty = false
	return t.worldMatrix
}
