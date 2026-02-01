package main

import (
	"encoding/json"
	"fmt"
	"kasir-api/database"
	"kasir-api/handlers"
	"kasir-api/repositories"
	"kasir-api/services"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/viper"
)

////////////////////////////////////////////////
// Configuration
////////////////////////////////////////////////

type Config struct {
	Port   string `mapstructure:"PORT"`
	DBConn string `mapstructure:"DB_CONN"`
}

func LoadConfig() (Config, error) {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if _, err := os.Stat(".env"); err == nil {
		viper.SetConfigFile(".env")
		_ = viper.ReadInConfig() // Ignore error if .env doesn't exist
	}

	config := Config{
		Port:   viper.GetString("PORT"),
		DBConn: viper.GetString("DB_CONN"),
	}

	// Fallback to os.Getenv if Viper fails to pick up from certain platforms
	if config.DBConn == "" {
		config.DBConn = os.Getenv("DB_CONN")
	}

	if config.Port == "" {
		config.Port = os.Getenv("PORT")
		if config.Port == "" {
			config.Port = "8080"
		}
	}

	return config, nil
}

////////////////////////////////////////////////

func main() {
	// 1. Load Configuration
	config, err := LoadConfig()
	if err != nil {
		log.Printf("Gagal load config: %v\n", err)
	}

	// DEBUG: Pastikan DB_CONN kebaca (jangan print password!)
	if config.DBConn == "" {
		log.Println("WARNING: DB_CONN is empty!")
	} else {
		log.Printf("DB_CONN found (length: %d)\n", len(config.DBConn))
	}

	// 2. Setup Database
	db, err := database.InitDB(config.DBConn)
	if err != nil {
		log.Fatal("Gagal inisialisasi database: ", err)
	}
	defer db.Close()

	// 3. Setup Layers (Wiring)
	// Repo -> Service -> Handler
	productRepo := repositories.NewProductRepository(db)
	productService := services.NewProductService(productRepo)
	productHandler := handlers.NewProductHandler(productService)
	categoryRepo := repositories.NewCategoryRepository(db)
	categoryService := services.NewCategoryService(categoryRepo)
	categoryHandler := handlers.NewCategoryHandler(categoryService)

	// 4. Register Routes
	// Root welcome message
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Welcome to Kasir API! silakan cek /health, /api/produk, atau /api/category",
		})
	})

	// Health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "OK",
			"message": "API Running",
		})
	})

	// Setup routes for Product (menggunakan Handler baru)
	http.HandleFunc("/api/produk", productHandler.HandleProducts)
	http.HandleFunc("/api/produk/", productHandler.HandleProductByID)

	// Setup routes for Category (menggunakan Handler baru)
	http.HandleFunc("/api/category", categoryHandler.HandleCategories)
	http.HandleFunc("/api/category/", categoryHandler.HandleCategoryByID)

	// 5. Start Server
	addr := "0.0.0.0:" + config.Port
	fmt.Printf("Server siap! Berjalan di: %s\n", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Printf("Waduh, server gagal jalan: %v\n", err)
	}
}
