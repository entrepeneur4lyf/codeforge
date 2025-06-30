package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/ml"
	"github.com/spf13/cobra"
)

// mlCmd represents the ml command for ML-related operations
var mlCmd = &cobra.Command{
	Use:   "ml",
	Short: "ML-powered code intelligence commands",
	Long: `ML commands provide access to machine learning features including:
- Smart code search using TD Learning
- Performance statistics and monitoring
- Configuration management
- Learning from user feedback`,
	Run: func(cmd *cobra.Command, args []string) {
		// Show ML status by default
		showMLStatus()
	},
}

// mlSearchCmd performs ML-enhanced code search
var mlSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Perform ML-enhanced code search",
	Long: `Search your codebase using ML-powered intelligence.
The system uses TD Learning and MCTS to find the most relevant code.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := strings.Join(args, " ")
		performMLSearch(query)
	},
}

// mlStatsCmd shows ML performance statistics
var mlStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show ML performance statistics",
	Long:  `Display detailed statistics about ML learning performance and efficiency.`,
	Run: func(cmd *cobra.Command, args []string) {
		showMLStats()
	},
}

// mlEnableCmd enables ML features
var mlEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable ML features",
	Long:  `Enable ML-powered code intelligence features.`,
	Run: func(cmd *cobra.Command, args []string) {
		enableML(true)
	},
}

// mlDisableCmd disables ML features
var mlDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable ML features",
	Long:  `Disable ML-powered code intelligence features.`,
	Run: func(cmd *cobra.Command, args []string) {
		enableML(false)
	},
}

// mlConfigCmd manages ML configuration
var mlConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage ML configuration",
	Long:  `View and modify ML configuration parameters.`,
	Run: func(cmd *cobra.Command, args []string) {
		showMLConfig()
	},
}

// mlLearnCmd simulates learning from user feedback
var mlLearnCmd = &cobra.Command{
	Use:   "learn [query] [feedback]",
	Short: "Simulate learning from user feedback",
	Long: `Simulate user feedback for testing ML learning capabilities.
Feedback should be a number between 0.0 and 1.0.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]
		feedback, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			fmt.Printf("❌ Invalid feedback value: %s (should be 0.0-1.0)\n", args[1])
			return
		}
		simulateMLLearning(query, feedback)
	},
}

func init() {
	// Add ML commands to root
	rootCmd.AddCommand(mlCmd)

	// Add subcommands to ml
	mlCmd.AddCommand(mlSearchCmd)
	mlCmd.AddCommand(mlStatsCmd)
	mlCmd.AddCommand(mlEnableCmd)
	mlCmd.AddCommand(mlDisableCmd)
	mlCmd.AddCommand(mlConfigCmd)
	mlCmd.AddCommand(mlLearnCmd)
}

// Command implementations

func showMLStatus() {
	service := ml.GetService()
	if service == nil {
		fmt.Println("🧠 ML Service: Not initialized")
		return
	}

	if service.IsEnabled() {
		fmt.Println("🧠 ML Service: ✅ Enabled and running")
		fmt.Println("   Features: TD Learning, Smart Search, Adaptive Context")
	} else {
		fmt.Println("🧠 ML Service: ⚠️  Disabled")
		fmt.Println("   Use 'codeforge ml enable' to activate ML features")
	}

	// Show quick stats
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats := service.GetStats(ctx)
	if stats != "" {
		fmt.Println("\n" + stats)
	}
}

func performMLSearch(query string) {
	service := ml.GetService()
	if service == nil || !service.IsEnabled() {
		fmt.Println("❌ ML Service not available. Use 'codeforge ml enable' to activate.")
		return
	}

	fmt.Printf("🔍 Searching with ML intelligence: %s\n", query)
	fmt.Println("⏳ Analyzing codebase...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	result := service.SmartSearch(ctx, query)
	duration := time.Since(start)

	if result == "" {
		fmt.Println("❌ No results found")
		return
	}

	fmt.Printf("⚡ Search completed in %v\n\n", duration)
	fmt.Println(result)

	// Simulate positive feedback for demonstration
	go service.LearnFromInteraction(ctx, query, []string{}, 0.8)
}

func showMLStats() {
	service := ml.GetService()
	if service == nil {
		fmt.Println("❌ ML Service not initialized")
		return
	}

	if !service.IsEnabled() {
		fmt.Println("⚠️  ML Service is disabled")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats := service.GetStats(ctx)
	if stats == "" {
		fmt.Println("❌ Unable to retrieve ML statistics")
		return
	}

	fmt.Println(stats)

	// Show TD Learning specific stats if available
	if tdStats := service.GetTDStats(); tdStats != nil {
		fmt.Println("\n## 🧠 TD Learning Details")
		fmt.Printf("**Lambda (λ):** %.2f\n", tdStats.Lambda)
		fmt.Printf("**Total Steps:** %d\n", tdStats.TotalSteps)
		fmt.Printf("**Average TD Error:** %.4f\n", tdStats.AverageTDError)
		fmt.Printf("**Active Traces:** %d\n", tdStats.ActiveTraces)
		fmt.Printf("**Learning Rate:** %.3f\n", tdStats.LearningRate)
		fmt.Printf("**Last Updated:** %s\n", tdStats.LastUpdateTime.Format("15:04:05"))
	}
}

func enableML(enable bool) {
	service := ml.GetService()
	if service == nil {
		fmt.Println("❌ ML Service not initialized")
		return
	}

	service.SetEnabled(enable)

	if enable {
		fmt.Println("✅ ML features enabled")
		fmt.Println("   TD Learning and smart search are now active")
	} else {
		fmt.Println("⚠️  ML features disabled")
		fmt.Println("   Falling back to traditional search methods")
	}
}

func showMLConfig() {
	service := ml.GetService()
	if service == nil {
		fmt.Println("❌ ML Service not initialized")
		return
	}

	fmt.Println("## ⚙️ ML Configuration")

	if service.IsEnabled() {
		fmt.Println("**Status:** ✅ Enabled")
	} else {
		fmt.Println("**Status:** ⚠️  Disabled")
	}

	// Show TD Learning configuration
	if tdStats := service.GetTDStats(); tdStats != nil {
		fmt.Println("\n**TD Learning Configuration:**")
		fmt.Printf("- Lambda (λ): %.2f\n", tdStats.Lambda)
		fmt.Printf("- Learning Rate: %.3f\n", tdStats.LearningRate)
		fmt.Printf("- Discount Factor: %.2f\n", tdStats.DiscountFactor)
		fmt.Printf("- Max Traces: %d\n", tdStats.MaxTraces)
		fmt.Printf("- Trace Threshold: %.3f\n", tdStats.TraceThreshold)
	}

	fmt.Println("\n**Available Commands:**")
	fmt.Println("- `codeforge ml enable/disable` - Toggle ML features")
	fmt.Println("- `codeforge ml search <query>` - ML-enhanced search")
	fmt.Println("- `codeforge ml stats` - Performance statistics")
	fmt.Println("- `codeforge ml learn <query> <feedback>` - Simulate learning")
}

func simulateMLLearning(query string, feedback float64) {
	service := ml.GetService()
	if service == nil || !service.IsEnabled() {
		fmt.Println("❌ ML Service not available")
		return
	}

	if feedback < 0.0 || feedback > 1.0 {
		fmt.Println("❌ Feedback must be between 0.0 and 1.0")
		return
	}

	fmt.Printf("🧠 Learning from feedback: query='%s', feedback=%.2f\n", query, feedback)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Simulate learning
	service.LearnFromInteraction(ctx, query, []string{}, feedback)

	fmt.Println("✅ Learning completed")

	// Show updated stats
	if tdStats := service.GetTDStats(); tdStats != nil {
		fmt.Printf("📊 Updated stats - Steps: %d, TD Error: %.4f\n",
			tdStats.TotalSteps, tdStats.AverageTDError)
	}
}
