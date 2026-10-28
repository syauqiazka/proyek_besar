package recipes

import (
	"fmt"
)

// Recipe defines a cleanup recipe (kategori B: reclaimable safe operations)
type Recipe struct {
	ID          string
	Name        string
	Description string
	Category    string // "cache", "logs", "temp", "docker"
	Command     string // SSH command template
	SafetyLevel string // "safe", "moderate" (all kategori B are pre-vetted)
	EstimateCmd string // command to estimate bytes before cleanup
	Reversible  bool   // false for kategori B (prinsip: no trash)
}

// AllRecipes returns all available cleanup recipes
var AllRecipes = []Recipe{
	{
		ID:          "apt-cache",
		Name:        "APT package cache",
		Description: "Hapus cache paket APT yang sudah diunduh (aman, bisa diunduh ulang kapan saja)",
		Category:    "cache",
		Command:     "sudo apt-get clean",
		SafetyLevel: "safe",
		EstimateCmd: "du -sb /var/cache/apt/archives 2>/dev/null | awk '{print $1}'",
		Reversible:  false,
	},
	{
		ID:          "journal-logs",
		Name:        "Journal logs lama",
		Description: "Hapus systemd journal log lebih dari 7 hari (log lama biasanya tidak diperlukan)",
		Category:    "logs",
		Command:     "sudo journalctl --vacuum-time=7d",
		SafetyLevel: "safe",
		EstimateCmd: "journalctl --disk-usage 2>/dev/null | grep -oP '\\d+\\.\\d+[MGK]' | head -1",
		Reversible:  false,
	},
	{
		ID:          "docker-unused-images",
		Name:        "Docker image tidak terpakai",
		Description: "Hapus Docker image yang tidak terkait dengan container aktif (dangling images)",
		Category:    "docker",
		Command:     "docker image prune -af --filter 'dangling=true'",
		SafetyLevel: "safe",
		EstimateCmd: "docker system df -v 2>/dev/null | grep 'Images space usage' | awk '{print $4}'",
		Reversible:  false,
	},
	{
		ID:          "docker-build-cache",
		Name:        "Docker build cache",
		Description: "Hapus cache build Docker (aman, akan dibuat ulang saat build berikutnya)",
		Category:    "docker",
		Command:     "docker builder prune -af",
		SafetyLevel: "safe",
		EstimateCmd: "docker system df -v 2>/dev/null | grep 'Build cache usage' | awk '{print $4}'",
		Reversible:  false,
	},
	{
		ID:          "npm-cache",
		Name:        "NPM global cache",
		Description: "Hapus cache NPM global (aman, package bisa diunduh ulang)",
		Category:    "cache",
		Command:     "npm cache clean --force",
		SafetyLevel: "safe",
		EstimateCmd: "du -sb ~/.npm 2>/dev/null | awk '{print $1}'",
		Reversible:  false,
	},
	{
		ID:          "tmp-old-files",
		Name:        "File /tmp lebih dari 7 hari",
		Description: "Hapus file di /tmp yang tidak diakses lebih dari 7 hari",
		Category:    "temp",
		Command:     "find /tmp -type f -atime +7 -delete 2>/dev/null",
		SafetyLevel: "moderate",
		EstimateCmd: "find /tmp -type f -atime +7 -exec du -cb {} + 2>/dev/null | tail -1 | awk '{print $1}'",
		Reversible:  false,
	},
	{
		ID:          "rotated-logs",
		Name:        "Log yang sudah dirotasi (.gz)",
		Description: "Hapus log terkompresi hasil rotasi logrotate (log lama sudah dirotasi)",
		Category:    "logs",
		Command:     "find /var/log -type f -name '*.gz' -mtime +30 -delete 2>/dev/null",
		SafetyLevel: "moderate",
		EstimateCmd: "find /var/log -type f -name '*.gz' -mtime +30 -exec du -cb {} + 2>/dev/null | tail -1 | awk '{print $1}'",
		Reversible:  false,
	},
}

// GetRecipeByID retrieves a recipe by ID
func GetRecipeByID(id string) (*Recipe, error) {
	for _, recipe := range AllRecipes {
		if recipe.ID == id {
			return &recipe, nil
		}
	}
	return nil, fmt.Errorf("recipe not found: %s", id)
}

// GetRecipesByCategory filters recipes by category
func GetRecipesByCategory(category string) []Recipe {
	var filtered []Recipe
	for _, recipe := range AllRecipes {
		if recipe.Category == category {
			filtered = append(filtered, recipe)
		}
	}
	return filtered
}
