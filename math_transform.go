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
func (self *Transform) Reset() {
	self.position = V2(0) // Default position (0, 0)
	self.scale = V2(1)    // Default scale (1, 1)
	self.offset = V2(0)   // Default offset (0, 0)
	self.origin = V2(0)   // Default origin (0, 0)
	self.rotation = 0.0   // Default rotation (0.0)
	// Mark as dirty to force world matrix recalculation next time Matrix() is called.
	self.isDirty = true
}

// Methods for Parent Hierarchy
// ----------------------------
// GetParentTransform returns the parent Transform or nil if there is no parent.
func (self *Transform) GetParentTransform() *Transform {
	return self.parent
}

// GetInitialParentTransform finds the topmost parent in the hierarchy.
// It iterates up the parent chain until it finds the root transform.
func (self *Transform) GetInitialParentTransform() *Transform {
	for self.parent != nil {
		self = self.parent
	}
	return self
}

// GetTransform returns this Transform.
// This method fulfills the Transformer interface.
func (self *Transform) GetTransform() *Transform {
	return self
}

// Transformation Properties
// -------------------------
// Origin returns the origin of the transform.
func (self *Transform) Origin() Vector {
	return self.origin
}

// SetOrigin updates the transform's origin and marks it as dirty.
func (self *Transform) SetOrigin(origin Vector) {
	self.isDirty = true
	self.origin = origin
}

// IsDirty checks if this transform or any of its parents are dirty.
// This recursive check is used for cache invalidation.
func (self *Transform) IsDirty() bool {
	// A single, recursive check for dirty state.
	if self.isDirty {
		return true
	}
	if self.parent != nil {
		return self.parent.IsDirty()
	}
	return false
}

// Position and Movement
// ---------------------
// SetPosition updates the position, preserving the world-space position
// by adjusting the local position based on the parent's inverse matrix.
func (self *Transform) SetPosition(position Vector) {
	self.isDirty = true
	if self.parent != nil {
		// Calculate the local position by transforming the world position by the parent's inverse matrix.
		worldToLocalMatrix := self.parent.Matrix()
		worldToLocalMatrix.Invert()
		self.position = position.Apply(worldToLocalMatrix)
	} else {
		// If there is no parent, the local position is the world position.
		self.position = position
	}
}

// Position returns the absolute position in world space.
// It calculates the world position by applying the transform's world matrix to the zero vector.
func (self *Transform) Position() Vector {
	return self.position
}

// Move translates the transform by the given vector(s).
func (self *Transform) Move(v ...Vector) {
	self.SetPosition(self.Position().Add(v...))
}

// Rotation
// --------
// SetRotation updates the rotation, preserving the world-space rotation
// by adjusting the local rotation based on the parent's rotation.
func (self *Transform) SetRotation(rotation float64) {
	self.isDirty = true
	if self.parent != nil {
		// Calculate the local rotation by subtracting the parent's world rotation.
		self.rotation = rotation - self.parent.Rotation()
	} else {
		// If there is no parent, the local rotation is the world rotation.
		self.rotation = rotation
	}
}

// Rotation returns the absolute rotation in world space.
func (self *Transform) Rotation() float64 {
	if self.parent == nil {
		return self.rotation
	}
	return self.rotation + self.parent.Rotation()
}

// Rotate adds to the current rotation.
func (self *Transform) Rotate(rotation float64) {
	self.isDirty = true
	self.rotation += rotation
}

// Scale
// -----
// SetScale updates the local scale.
func (self *Transform) SetScale(scale Vector) {
	self.isDirty = true
	self.scale = scale
}

// Scale returns the absolute scale in world space by multiplying with the parent's scale.
func (self *Transform) Scale() Vector {
	if self.parent == nil {
		return self.scale
	}
	return self.scale.Scale(self.parent.Scale())
}

// AddScale adds to the current local scale.
func (self *Transform) AddScale(add ...Vector) {
	self.isDirty = true
	self.scale = self.scale.Add(add...)
}

// Offset
// ------
// SetOffset updates the local offset.
func (self *Transform) SetOffset(offset Vector) {
	self.isDirty = true
	self.offset = offset
}

// Offset returns the local offset.
func (self *Transform) Offset() Vector {
	return self.offset
}

// Transform Modifiers
// -------------------
// Abs returns a new transform with the absolute world properties of this transform,
// effectively disconnecting it from its parent hierarchy.
func (self *Transform) Abs() Transform {
	abs := *T()
	abs.SetPosition(self.Position())
	abs.SetRotation(self.Rotation())
	abs.SetScale(self.Scale())
	abs.SetOffset(self.Offset())
	abs.SetOrigin(self.Origin())
	return abs
}

// Rel returns a copy of the transform with its parent set to nil.
func (self *Transform) Rel() Transform {
	rel := *self
	rel.parent = nil
	return rel
}

// Parent Management
// -----------------
// Connected returns true if the transform has a parent.
func (self *Transform) Connected() bool {
	return self.parent != nil
}

// Replace updates this transform's local properties to match the world properties
// of another transform.
func (self *Transform) Replace(new Transformer) {
	nt := new.GetTransform()
	self.SetPosition(nt.Position())
	self.SetOffset(nt.Offset())
	self.SetRotation(nt.Rotation())
	self.SetOrigin(nt.Origin())
	self.SetScale(nt.Scale())
}

// Connect establishes a parent-child relationship, preserving the object's
// world space transform.
func (self *Transform) Connect(parent Transformer) {
	if parent == nil {
		return
	}
	// Store the current world properties before connecting to the parent.
	worldPos := self.Position()
	worldRot := self.Rotation()
	worldScale := self.Scale()
	worldOffset := self.Offset()
	worldOrigin := self.Origin()
	// Set the new parent and mark the transform as dirty.
	self.parent = parent.GetTransform()
	self.isDirty = true
	// Re-apply the stored world properties, which will internally
	// calculate the new local properties based on the new parent.
	self.SetPosition(worldPos)
	self.SetRotation(worldRot)
	self.SetScale(worldScale)
	self.SetOffset(worldOffset)
	self.SetOrigin(worldOrigin)
}

// Disconnect removes the parent relationship, making the transform absolute.
// It does this by creating a new absolute transform and overwriting the current one.
func (self *Transform) Disconnect() {
	if self.parent == nil {
		return
	}
	*self = self.Abs()
}

// Matrix Operations
// -----------------
// MatrixForParenting returns matrices for child positioning.
// It returns the world matrix without the origin offset and its inverse.
// It ensures the world matrix is up-to-date by calling Matrix() if needed.
func (self *Transform) MatrixForParenting() (Matrix, Matrix) {
	if self.isDirty {
		self.Matrix() // Ensure the world matrix is computed and cached.
	}
	return self.parentMatrix, self.parentInverted
}

// Matrix computes the full world transformation matrix for this node.
// The result is cached to avoid repeated calculations.
func (self *Transform) Matrix() Matrix {
	if !self.IsDirty() {
		return self.worldMatrix
	}
	// Calculate the local transformation matrix.
	localMatrix := Matrix{} // Scale first.
	localMatrix.Scale(self.scale.X, self.scale.Y)
	// Then move and rotate.
	localMatrix.Translate(
		-self.offset.X*self.scale.X,
		-self.offset.Y*self.scale.Y,
	)
	localMatrix.Rotate(float64(self.rotation))
	// Move to the absolute position.
	localMatrix.Translate(self.position.X, self.position.Y)
	// Save this local matrix as the parent matrix for children, without the origin offset.
	self.parentMatrix = localMatrix
	self.parentInverted = localMatrix
	self.parentInverted.Invert()
	// And finally move to the origin offset
	if !self.origin.IsZero() {
		localMatrix.Translate(-self.origin.X, -self.origin.Y)
	}
	// If there's a parent, combine this local matrix with the parent's world matrix.
	if self.parent != nil {
		parentMatrix := self.parent.Matrix()
		localMatrix.Concat(parentMatrix)
	}
	self.worldMatrix = localMatrix
	self.isDirty = false
	return self.worldMatrix
}
