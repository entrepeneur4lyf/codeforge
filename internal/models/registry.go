package models

import (
	"fmt"
	"sync"
	"time"
)

// ModelRegistry manages the canonical model registry
type ModelRegistry struct {
	models map[CanonicalModelID]*CanonicalModel
	mutex  sync.RWMutex
}

// NewModelRegistry creates a new model registry
func NewModelRegistry() *ModelRegistry {
	registry := &ModelRegistry{
		models: make(map[CanonicalModelID]*CanonicalModel),
	}

	// Initialize with hardcoded frontier models
	registry.initializeFrontierModels()

	return registry
}

// GetModel retrieves a model by canonical ID
func (mr *ModelRegistry) GetModel(id CanonicalModelID) (*CanonicalModel, bool) {
	mr.mutex.RLock()
	defer mr.mutex.RUnlock()

	model, exists := mr.models[id]
	return model, exists
}

// GetModelByProvider retrieves a model by provider and provider model ID
func (mr *ModelRegistry) GetModelByProvider(providerID ProviderID, providerModelID string) (*CanonicalModel, bool) {
	mr.mutex.RLock()
	defer mr.mutex.RUnlock()

	for _, model := range mr.models {
		if mapping, exists := model.Providers[providerID]; exists {
			if mapping.ProviderModelID == providerModelID {
				return model, true
			}
		}
	}

	return nil, false
}

// ListModels returns all models in the registry
func (mr *ModelRegistry) ListModels() []*CanonicalModel {
	mr.mutex.RLock()
	defer mr.mutex.RUnlock()

	models := make([]*CanonicalModel, 0, len(mr.models))
	for _, model := range mr.models {
		models = append(models, model)
	}

	return models
}

// ListModelsByProvider returns all models available for a specific provider
func (mr *ModelRegistry) ListModelsByProvider(providerID ProviderID) []*CanonicalModel {
	mr.mutex.RLock()
	defer mr.mutex.RUnlock()

	var models []*CanonicalModel
	for _, model := range mr.models {
		if mapping, exists := model.Providers[providerID]; exists && mapping.Available {
			models = append(models, model)
		}
	}

	return models
}

// AddModel adds a new model to the registry
func (mr *ModelRegistry) AddModel(model *CanonicalModel) error {
	mr.mutex.Lock()
	defer mr.mutex.Unlock()

	if model.ID == "" {
		return fmt.Errorf("model ID cannot be empty")
	}

	if _, exists := mr.models[model.ID]; exists {
		return fmt.Errorf("model %s already exists", model.ID)
	}

	now := time.Now()
	model.CreatedAt = now
	model.UpdatedAt = now

	mr.models[model.ID] = model
	return nil
}

// UpdateModel updates an existing model in the registry
func (mr *ModelRegistry) UpdateModel(model *CanonicalModel) error {
	mr.mutex.Lock()
	defer mr.mutex.Unlock()

	if _, exists := mr.models[model.ID]; !exists {
		return fmt.Errorf("model %s does not exist", model.ID)
	}

	model.UpdatedAt = time.Now()
	mr.models[model.ID] = model
	return nil
}

// UpdateModelAvailability updates the availability of a model for a specific provider
func (mr *ModelRegistry) UpdateModelAvailability(modelID CanonicalModelID, providerID ProviderID, available bool) error {
	mr.mutex.Lock()
	defer mr.mutex.Unlock()

	model, exists := mr.models[modelID]
	if !exists {
		return fmt.Errorf("model %s does not exist", modelID)
	}

	if mapping, exists := model.Providers[providerID]; exists {
		mapping.Available = available
		mapping.LastChecked = time.Now()
		model.Providers[providerID] = mapping
		model.UpdatedAt = time.Now()
	}

	return nil
}

// GetProviderModelID gets the provider-specific model ID for a canonical model
func (mr *ModelRegistry) GetProviderModelID(modelID CanonicalModelID, providerID ProviderID) (string, error) {
	mr.mutex.RLock()
	defer mr.mutex.RUnlock()

	model, exists := mr.models[modelID]
	if !exists {
		return "", fmt.Errorf("model %s does not exist", modelID)
	}

	mapping, exists := model.Providers[providerID]
	if !exists {
		return "", fmt.Errorf("model %s is not available on provider %s", modelID, providerID)
	}

	if !mapping.Available {
		return "", fmt.Errorf("model %s is not currently available on provider %s", modelID, providerID)
	}

	return mapping.ProviderModelID, nil
}

// initializeFrontierModels initializes the registry with hardcoded frontier models
// Based on Cline's hardcoded model definitions for instant access
func (mr *ModelRegistry) initializeFrontierModels() {
	now := time.Now()

	// Copy models from the global CanonicalModels map
	for id, model := range CanonicalModels {
		// Create a copy to avoid shared references
		modelCopy := *model
		modelCopy.CreatedAt = now
		modelCopy.UpdatedAt = now
		mr.models[id] = &modelCopy
	}
}
