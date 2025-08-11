package managers

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

type ImageBufferManager struct {
	buffers map[string]*ebiten.Image
	mutex   sync.RWMutex
}

var (
	defaultManager = &ImageBufferManager{
		buffers: make(map[string]*ebiten.Image, 100), // Initial capacity hint
	}
)

// GetOrCreateImageBuffer returns new or existing image
func GetOrCreateImageBuffer(id string, width, height int) *ebiten.Image {
	if !isValidDimensions(width, height) {
		return nil
	}

	defaultManager.mutex.Lock()
	defer defaultManager.mutex.Unlock()

	buffer, exists := defaultManager.buffers[id]
	if exists {
		if buffer.Bounds().Dx() != width || buffer.Bounds().Dy() != height {
			buffer.Deallocate() // Clean up old image
			buffer = ebiten.NewImage(width, height)
			defaultManager.buffers[id] = buffer
		}
		return buffer
	}

	buffer = ebiten.NewImage(width, height)
	defaultManager.buffers[id] = buffer
	return buffer
}

func Get(id string) *ebiten.Image {
	return defaultManager.buffers[id]
}

// TotalImageBuffers returns total cached images in buffers
func TotalImageBuffers() int {
	defaultManager.mutex.RLock()
	defer defaultManager.mutex.RUnlock()
	return len(defaultManager.buffers)
}

// RemoveImageBuffer is used to remove unused image
func RemoveImageBuffer(id string) {
	defaultManager.mutex.Lock()
	defer defaultManager.mutex.Unlock()

	if buffer, exists := defaultManager.buffers[id]; exists {
		buffer.Deallocate() // Clean up before removing
		delete(defaultManager.buffers, id)
	}
}

// ClearAllImageBuffers is used to remove all existing images
func ClearAllImageBuffers() {
	defaultManager.mutex.Lock()
	defer defaultManager.mutex.Unlock()

	for _, buffer := range defaultManager.buffers {
		buffer.Deallocate() // Clean up all images
	}
	clear(defaultManager.buffers)
}

func isValidDimensions(width, height int) bool {
	return width > 0 && height > 0
}

// ResizeBuffer resizes an existing buffer or creates a new one
func ResizeBuffer(id string, newWidth, newHeight int) *ebiten.Image {
	if !isValidDimensions(newWidth, newHeight) {
		return nil
	}
	return GetOrCreateImageBuffer(id, newWidth, newHeight)
}

// HasBuffer checks if a buffer exists
func HasBuffer(id string) bool {
	defaultManager.mutex.RLock()
	defer defaultManager.mutex.RUnlock()
	_, exists := defaultManager.buffers[id]
	return exists
}
