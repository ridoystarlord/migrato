package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ridoystarlord/migrato/database"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var studioCmd = &cobra.Command{
	Use:   "studio",
	Short: "Launch web-based database browser",
	Long: `Launch Migrato Studio - a web-based database browser for viewing and editing table data.

This opens a web interface in your browser where you can:
- Browse all tables in your database
- View table data with pagination
- Search and filter data
- Edit data inline (optional)

The interface will be available at http://localhost:8080 by default.`,
	Run: func(cmd *cobra.Command, args []string) {
		port := viper.GetString("studio.port")
		if port == "" {
			port = "8080"
		}

		fmt.Printf("🚀 Starting Migrato Studio on http://localhost:%s\n", port)
		fmt.Println("Press Ctrl+C to stop the server")

		// Start the web server
		if err := startStudioServer(port); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(studioCmd)
	
	// Add studio-specific flags
	studioCmd.Flags().String("port", "8080", "Port to run the web server on")
	viper.BindPFlag("studio.port", studioCmd.Flags().Lookup("port"))
}

func startStudioServer(port string) error {
	// Create web server
	server := &StudioServer{
		port: port,
	}

	// Setup routes
	http.HandleFunc("/", server.handleIndex)
	http.HandleFunc("/api/tables", server.handleTables)
	http.HandleFunc("/api/relationships", server.handleRelationships)
	http.HandleFunc("/api/table/", server.handleTableData)
	http.HandleFunc("/api/update/", server.handleUpdateData)
	http.HandleFunc("/api/export/", server.handleExportData)
	http.HandleFunc("/api/import/", server.handleImportData)
	http.HandleFunc("/static/", server.handleStatic)

	// Start server
	return http.ListenAndServe(":"+port, nil)
}

// StudioServer handles the web interface
type StudioServer struct {
	port string
}

// TableInfo represents table metadata
type TableInfo struct {
	Name    string `json:"name"`
	Columns []ColumnInfo `json:"columns"`
}

// ColumnInfo represents column metadata
type ColumnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Default  *string `json:"default,omitempty"`
}

// TableData represents paginated table data
type TableData struct {
	Data  []map[string]interface{} `json:"data"`
	Total int                      `json:"total"`
	Page  int                      `json:"page"`
	Limit int                      `json:"limit"`
}

// Relationship represents a foreign key relationship between tables
type Relationship struct {
	SourceTable    string `json:"source_table"`
	SourceColumn   string `json:"source_column"`
	TargetTable    string `json:"target_table"`
	TargetColumn   string `json:"target_column"`
	ConstraintName string `json:"constraint_name"`
}

func (s *StudioServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Serve the main HTML page
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Migrato Studio - Database Browser</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://cdn.jsdelivr.net/npm/mermaid@10.6.1/dist/mermaid.min.js"></script>
    <script>
        tailwind.config = {
            theme: {
                extend: {
                    colors: {
                        primary: {
                            50: '#eff6ff',
                            500: '#3b82f6',
                            600: '#2563eb',
                            700: '#1d4ed8',
                        }
                    }
                }
            }
        }
    </script>
    <style>
        /* Custom scrollbar */
        ::-webkit-scrollbar {
            width: 8px;
            height: 8px;
        }
        ::-webkit-scrollbar-track {
            background: #f1f5f9;
        }
        ::-webkit-scrollbar-thumb {
            background: #cbd5e1;
            border-radius: 4px;
        }
        ::-webkit-scrollbar-thumb:hover {
            background: #94a3b8;
        }
        
        /* Smooth animations */
        .fade-in {
            animation: fadeIn 0.3s ease-in-out;
        }
        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(10px); }
            to { opacity: 1; transform: translateY(0); }
        }
        
        .slide-in {
            animation: slideIn 0.3s ease-out;
        }
        @keyframes slideIn {
            from { transform: translateX(-20px); opacity: 0; }
            to { transform: translateX(0); opacity: 1; }
        }
        
        /* Loading animation */
        .loading-dots {
            display: inline-block;
        }
        .loading-dots::after {
            content: '';
            animation: dots 1.5s steps(5, end) infinite;
        }
        @keyframes dots {
            0%, 20% { content: ''; }
            40% { content: '.'; }
            60% { content: '..'; }
            80%, 100% { content: '...'; }
        }
        
        /* Custom background for alternate rows */
        .bg-slate-750 {
            background-color: #1e293b;
        }
        
        /* Sidebar collapse animation */
        .sidebar-collapsed {
            transform: translateX(-100%);
            width: 0 !important;
            overflow: hidden;
        }
        
        /* Hide sidebar content when collapsed */
        .sidebar-collapsed * {
            opacity: 0;
            pointer-events: none;
        }
        
        /* Overlay for mobile */
        .sidebar-overlay {
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background-color: rgba(0, 0, 0, 0.5);
            z-index: 40;
            opacity: 0;
            visibility: hidden;
            transition: all 0.3s ease-in-out;
        }
        
        .sidebar-overlay.active {
            opacity: 1;
            visibility: visible;
        }
        
        /* Mobile sidebar positioning */
        @media (max-width: 1024px) {
            #sidebar {
                position: fixed;
                top: 4rem;
                left: 0;
                bottom: 0;
                z-index: 50;
                transform: translateX(-100%);
            }
            
            #sidebar.sidebar-open {
                transform: translateX(0);
            }
        }
        
        /* Light mode styles */
        body.light-mode {
            background-color: #f8fafc;
            color: #1e293b;
        }
        
        body.light-mode nav {
            background-color: #ffffff;
            border-bottom-color: #e2e8f0;
        }
        
        body.light-mode #sidebar {
            background-color: #ffffff;
            border-right-color: #e2e8f0;
        }
        
        body.light-mode #sidebar h3,
        body.light-mode #sidebar p {
            color: #1e293b;
        }
        
        body.light-mode #sidebar .text-slate-400 {
            color: #64748b;
        }
        
        body.light-mode main {
            background-color: #f8fafc;
        }
        
        body.light-mode .bg-slate-800 {
            background-color: #ffffff;
        }
        
        body.light-mode .border-slate-700 {
            border-color: #e2e8f0;
        }
        
        body.light-mode .text-white {
            color: #1e293b;
        }
        
        body.light-mode .text-slate-300 {
            color: #475569;
        }
        
        body.light-mode .text-slate-400 {
            color: #64748b;
        }
        
        body.light-mode .bg-slate-700 {
            background-color: #f1f5f9;
        }
        
        body.light-mode .bg-slate-750 {
            background-color: #f8fafc;
        }
        
        body.light-mode .hover\:bg-slate-700:hover {
            background-color: #e2e8f0;
        }
        
        body.light-mode .border-slate-600 {
            border-color: #cbd5e1;
        }
        
        body.light-mode input {
            background-color: #ffffff;
            border-color: #cbd5e1;
            color: #1e293b;
        }
        
        body.light-mode input::placeholder {
            color: #94a3b8;
        }
        
        body.light-mode .text-slate-500 {
            color: #64748b;
        }
        
        body.light-mode .text-red-400 {
            color: #f87171;
        }
        
        body.light-mode .text-blue-400 {
            color: #3b82f6;
        }
        
        body.light-mode .text-green-400 {
            color: #10b981;
        }
        
        body.light-mode .text-purple-400 {
            color: #8b5cf6;
        }
        
        /* Tab styles */
        .tab-btn {
            @apply px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 flex items-center;
            @apply bg-transparent text-slate-400 hover:text-white hover:bg-slate-700;
        }
        .tab-btn.active {
            @apply bg-slate-700 text-white;
        }
        .tab-content {
            display: none;
        }
        .tab-content.active {
            display: flex;
        }
    </style>
</head>
<body class="bg-slate-900 text-white h-screen overflow-hidden">
    <!-- Top Navigation Bar -->
    <nav class="bg-slate-800 border-b border-slate-700 h-16 flex items-center justify-between px-6">
        <div class="flex items-center space-x-4">
            <button id="sidebarToggle" class="p-2 rounded-lg hover:bg-slate-700 transition-colors text-slate-400 hover:text-white">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"></path>
                </svg>
            </button>
            <div>
                <h1 class="text-xl font-bold text-white">Migrato Studio</h1>
                <p class="text-slate-400 text-sm">Database Browser</p>
            </div>
        </div>
        <div class="flex items-center space-x-4">
            <button id="themeToggle" class="p-2 rounded-lg hover:bg-slate-700 transition-colors text-slate-400 hover:text-white">
                <svg id="themeIcon" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"></path>
                </svg>
            </button>
            <div class="flex items-center space-x-2 text-slate-400">
                <div class="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
                <span class="text-sm">Connected</span>
            </div>
        </div>
    </nav>
    
    <!-- Main Content Area -->
    <div class="flex h-[calc(100vh-4rem)]">
        <!-- Sidebar -->
        <aside id="sidebar" class="w-80 bg-slate-800 border-r border-slate-700 flex flex-col transition-all duration-300 ease-in-out">
            <div class="p-6 border-b border-slate-700">
                <div class="flex items-center justify-between">
                    <div>
                        <h3 class="text-lg font-semibold text-white mb-2">Tables</h3>
                        <p class="text-slate-400 text-sm">Select a table to view data</p>
                    </div>
                    <button id="sidebarClose" class="p-1 rounded hover:bg-slate-700 transition-colors text-slate-400 hover:text-white lg:hidden">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                        </svg>
                    </button>
                </div>
            </div>
            <div class="flex-1 overflow-y-auto">
                <div id="tableList" class="p-4 space-y-1">
                    <div class="text-slate-400 italic text-sm loading-dots">Loading tables</div>
                </div>
            </div>
        </aside>
        
        <!-- Main Content Area -->
        <main class="flex-1 bg-slate-900 overflow-hidden">
            <!-- Tabs -->
            <div class="bg-slate-800 border-b border-slate-700">
                <div class="flex space-x-1 p-4">
                    <button id="data-tab" class="tab-btn active" onclick="showTab('data')">
                        <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
                        </svg>
                        Data Browser
                    </button>
                    <button id="relationships-tab" class="tab-btn" onclick="showTab('relationships')">
                        <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path>
                        </svg>
                        Relationships
                    </button>
                </div>
            </div>
            
            <!-- Data Browser Tab -->
            <div id="data-view" class="tab-content active h-full flex flex-col">
                <div id="tableView" class="h-full flex flex-col">
                    <div class="flex-1 flex items-center justify-center">
                        <div class="text-center">
                            <div class="text-6xl mb-6 opacity-50">DB</div>
                            <h2 class="text-2xl font-semibold text-white mb-3">Welcome to Migrato Studio</h2>
                            <p class="text-slate-400 text-lg">Select a table from the sidebar to explore your data</p>
                            <div class="mt-8 flex items-center justify-center space-x-4 text-slate-500">
                                <div class="flex items-center space-x-2">
                                    <div class="w-2 h-2 bg-blue-500 rounded-full"></div>
                                    <span class="text-sm">Real-time data</span>
                                </div>
                                <div class="flex items-center space-x-2">
                                    <div class="w-2 h-2 bg-green-500 rounded-full"></div>
                                    <span class="text-sm">Fast search</span>
                                </div>
                                <div class="flex items-center space-x-2">
                                    <div class="w-2 h-2 bg-purple-500 rounded-full"></div>
                                    <span class="text-sm">Interactive</span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            
            <!-- Relationships Tab -->
            <div id="relationships-view" class="tab-content hidden h-full flex flex-col">
                <div class="flex-1 p-6">
                    <div class="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                        <div class="flex items-center justify-between mb-4">
                            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
                                Table Relationships
                            </h2>
                            <div class="flex space-x-2">
                                <button id="mermaid-view" class="btn btn-primary">Mermaid</button>
                                <button id="graph-view" class="btn btn-secondary">Interactive Graph</button>
                                <button id="tree-view" class="btn btn-secondary">Tree View</button>
                            </div>
                        </div>
                        
                        <div id="relationship-container" class="border rounded-lg p-4 min-h-96">
                            <div class="text-center text-gray-500 dark:text-gray-400">
                                <div class="text-4xl mb-4">🔗</div>
                                <p>Loading relationships...</p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </main>
    </div>
    
    <!-- Mobile overlay -->
    <div id="sidebarOverlay" class="sidebar-overlay lg:hidden"></div>
    
    <script src="/static/app.js"></script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (s *StudioServer) handleTables(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get database pool
	pool, err := s.getPool()
	if err != nil {
		http.Error(w, "Database connection failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := context.Background()
	// Query to get all tables
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		http.Error(w, "Failed to query tables: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			http.Error(w, "Failed to scan table name: "+err.Error(), http.StatusInternalServerError)
			return
		}
		tables = append(tables, tableName)
	}

	response := map[string]interface{}{
		"tables": tables,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *StudioServer) handleRelationships(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get database pool
	pool, err := s.getPool()
	if err != nil {
		http.Error(w, "Database connection failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := context.Background()

	// Query to get foreign key relationships
	query := `
		SELECT 
			tc.table_name as source_table,
			kcu.column_name as source_column,
			ccu.table_name as target_table,
			ccu.column_name as target_column,
			tc.constraint_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu 
			ON tc.constraint_name = kcu.constraint_name
		JOIN information_schema.constraint_column_usage ccu 
			ON ccu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
		AND tc.table_schema = 'public'
		AND ccu.table_schema = 'public'
		ORDER BY tc.table_name, kcu.column_name
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		http.Error(w, "Failed to query relationships: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var relationships []Relationship
	for rows.Next() {
		var rel Relationship
		err := rows.Scan(&rel.SourceTable, &rel.SourceColumn, &rel.TargetTable, &rel.TargetColumn, &rel.ConstraintName)
		if err != nil {
			http.Error(w, "Failed to scan relationship: "+err.Error(), http.StatusInternalServerError)
			return
		}
		relationships = append(relationships, rel)
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(relationships)
}

func (s *StudioServer) handleTableData(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract table name from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/table/")
	if path == "" {
		http.Error(w, "Table name required", http.StatusBadRequest)
		return
	}

	// Get query parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 1000 {
		limit = 50
	}

	search := r.URL.Query().Get("search")
	offset := (page - 1) * limit

	// Get database pool
	pool, err := s.getPool()
	if err != nil {
		http.Error(w, "Database connection failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := context.Background()

	// Build query with search
	var query string
	var args []interface{}
	
	// Validate table name to prevent SQL injection
	if !isValidTableName(path) {
		http.Error(w, "Invalid table name", http.StatusBadRequest)
		return
	}

	if search != "" {
		// For search, we'll use a simple LIKE query across all text columns
		// This is a simplified approach - in production you might want more sophisticated search
		query = "SELECT * FROM \"" + path + "\" WHERE "
		
		// Get column names first
		colQuery := "SELECT column_name, data_type FROM information_schema.columns WHERE table_name = $1 AND table_schema = 'public'"
		colRows, err := pool.Query(ctx, colQuery, path)
		if err != nil {
			http.Error(w, "Failed to get column info: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer colRows.Close()

		var searchConditions []string
		argIndex := 1
		
		for colRows.Next() {
			var colName, colType string
			if scanErr := colRows.Scan(&colName, &colType); scanErr != nil {
				continue
			}
			
			// Only search in text-like columns
			if strings.Contains(strings.ToLower(colType), "char") || 
			   strings.Contains(strings.ToLower(colType), "text") {
				searchConditions = append(searchConditions, "\""+colName+"\" ILIKE $"+strconv.Itoa(argIndex))
				args = append(args, "%"+search+"%")
				argIndex++
			}
		}
		
		if len(searchConditions) > 0 {
			query += strings.Join(searchConditions, " OR ")
		} else {
			query = "SELECT * FROM \"" + path + "\""
		}
	} else {
		query = "SELECT * FROM \"" + path + "\""
	}

	// Add pagination
	query += " LIMIT " + strconv.Itoa(limit) + " OFFSET " + strconv.Itoa(offset)

	// Execute query
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		http.Error(w, "Failed to query table data: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Get column names
	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columns[i] = string(fd.Name)
	}

	// Scan data
	var data []map[string]interface{}
	for rows.Next() {
		// Create a slice of interface{} to hold the values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if scanErr := rows.Scan(valuePtrs...); scanErr != nil {
			http.Error(w, "Failed to scan row: "+scanErr.Error(), http.StatusInternalServerError)
			return
		}

		// Convert to map
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if val == nil {
				row[col] = nil
			} else {
				row[col] = val
			}
		}
		data = append(data, row)
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM \"" + path + "\""
	if search != "" {
		// Use the same search conditions for count
		countQuery = "SELECT COUNT(*) FROM \"" + path + "\" WHERE "
		var countConditions []string
		searchPart := strings.TrimPrefix(query, "SELECT * FROM \""+path+"\" WHERE ")
		if searchPart != query { // If we have search conditions
			for _, condition := range strings.Split(searchPart, " OR ") {
				if strings.Contains(condition, "ILIKE") {
					countConditions = append(countConditions, condition)
				}
			}
		}
		if len(countConditions) > 0 {
			countQuery += strings.Join(countConditions, " OR ")
		} else {
			countQuery = "SELECT COUNT(*) FROM \"" + path + "\""
		}
	}
	
	err = pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		// If count fails, just use the data length
		total = len(data)
	}

	response := TableData{
		Data:  data,
		Total: total,
		Page:  page,
		Limit: limit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *StudioServer) handleUpdateData(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract table name from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/update/")
	if path == "" {
		http.Error(w, "Table name required", http.StatusBadRequest)
		return
	}

	// Validate table name to prevent SQL injection
	if !isValidTableName(path) {
		http.Error(w, "Invalid table name", http.StatusBadRequest)
		return
	}

	// Parse request body
	var updateRequest struct {
		RowID   string                 `json:"row_id"`
		IDValue string                 `json:"id_value"`
		Data    map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get database pool
	pool, err := s.getPool()
	if err != nil {
		http.Error(w, "Database connection failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := context.Background()

	// Validate the data against table schema
	if err := s.validateUpdateData(ctx, pool, path, updateRequest.Data); err != nil {
		http.Error(w, "Validation failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Build UPDATE query
	query, args, err := s.buildUpdateQuery(path, updateRequest.RowID, updateRequest.IDValue, updateRequest.Data)
	if err != nil {
		http.Error(w, "Failed to build update query: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Execute update
	result, err := pool.Exec(ctx, query, args...)
	if err != nil {
		http.Error(w, "Failed to update data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if any rows were affected
	if result.RowsAffected() == 0 {
		http.Error(w, "No rows were updated", http.StatusNotFound)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"success":     true,
		"message":     "Data updated successfully",
		"rows_affected": result.RowsAffected(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *StudioServer) getPool() (*pgxpool.Pool, error) {
	return database.GetPool()
}

func (s *StudioServer) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Serve static files (CSS, JS)
	path := r.URL.Path[8:] // Remove "/static/" prefix
	
	switch path {
	case "style.css":
		w.Header().Set("Content-Type", "text/css")
		w.Write([]byte(`/* Tailwind CSS is loaded via CDN */`))
	case "app.js":
		w.Header().Set("Content-Type", "application/javascript")
		// Read the external JavaScript file
		jsContent, err := os.ReadFile("static/app.js")
		if err != nil {
			http.Error(w, "JavaScript file not found", http.StatusNotFound)
			return
		}
		w.Write(jsContent)
	default:
		http.NotFound(w, r)
	}
}

// isValidTableName validates that the table name is safe for SQL queries
func isValidTableName(tableName string) bool {
	// Check if table name is empty or too long
	if tableName == "" || len(tableName) > 63 {
		return false
	}
	
	// Check if table name contains only alphanumeric characters and underscores
	for _, char := range tableName {
		if !((char >= 'a' && char <= 'z') || 
			 (char >= 'A' && char <= 'Z') || 
			 (char >= '0' && char <= '9') || 
			 char == '_') {
			return false
		}
	}
	
	// Check if table name doesn't start with a number
	if len(tableName) > 0 && tableName[0] >= '0' && tableName[0] <= '9' {
		return false
	}
	
	return true
}

func (s *StudioServer) validateUpdateData(ctx context.Context, pool *pgxpool.Pool, tableName string, data map[string]interface{}) error {
	// Get table schema to validate data types
	query := `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns 
		WHERE table_name = $1 AND table_schema = 'public'
		ORDER BY ordinal_position`

	rows, err := pool.Query(ctx, query, tableName)
	if err != nil {
		return fmt.Errorf("failed to get table schema: %w", err)
	}
	defer rows.Close()

	columnInfo := make(map[string]struct {
		DataType    string
		IsNullable  string
		HasDefault  bool
	})

	for rows.Next() {
		var colName, dataType, isNullable pgtype.Text
		var columnDefault pgtype.Text
		if err := rows.Scan(&colName, &dataType, &isNullable, &columnDefault); err != nil {
			return fmt.Errorf("failed to scan column info: %w", err)
		}
		columnInfo[colName.String] = struct {
			DataType    string
			IsNullable  string
			HasDefault  bool
		}{
			DataType:   dataType.String,
			IsNullable: isNullable.String,
			HasDefault: columnDefault.Valid,
		}
	}

	// Validate each field in the update data
	for fieldName, value := range data {
		colInfo, exists := columnInfo[fieldName]
		if !exists {
			return fmt.Errorf("column '%s' does not exist in table '%s'", fieldName, tableName)
		}

		// Check if value is null
		if value == nil {
			if colInfo.IsNullable == "NO" && !colInfo.HasDefault {
				return fmt.Errorf("column '%s' cannot be null", fieldName)
			}
			continue
		}

		// Basic type validation
		if err := s.validateFieldType(fieldName, value, colInfo.DataType); err != nil {
			return err
		}
	}

	return nil
}

func (s *StudioServer) validateFieldType(fieldName string, value interface{}, expectedType string) error {
	switch expectedType {
	case "integer", "bigint", "smallint":
		switch v := value.(type) {
		case float64:
			if v != float64(int64(v)) {
				return fmt.Errorf("column '%s' expects integer, got float", fieldName)
			}
		case int, int64, int32:
			// Valid
		default:
			return fmt.Errorf("column '%s' expects integer, got %T", fieldName, value)
		}
	case "numeric", "decimal", "real", "double precision":
		switch value.(type) {
		case float64, int, int64, int32:
			// Valid
		default:
			return fmt.Errorf("column '%s' expects numeric, got %T", fieldName, value)
		}
	case "boolean":
		switch value.(type) {
		case bool:
			// Valid
		default:
			return fmt.Errorf("column '%s' expects boolean, got %T", fieldName, value)
		}
	case "text", "character varying", "character", "uuid", "date", "timestamp", "timestamptz":
		switch value.(type) {
		case string:
			// Valid
		default:
			return fmt.Errorf("column '%s' expects text, got %T", fieldName, value)
		}
	}
	return nil
}

func (s *StudioServer) buildUpdateQuery(tableName, rowID, idValue string, data map[string]interface{}) (string, []interface{}, error) {
	if len(data) == 0 {
		return "", nil, fmt.Errorf("no data to update")
	}

	// Build SET clause
	setClause := make([]string, 0, len(data))
	args := make([]interface{}, 0, len(data)+1)
	argIndex := 1

	for fieldName, value := range data {
		setClause = append(setClause, fmt.Sprintf("%s = $%d", fieldName, argIndex))
		args = append(args, value)
		argIndex++
	}

	// Add WHERE clause
	whereClause := fmt.Sprintf("%s = $%d", rowID, argIndex)
	args = append(args, idValue)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", tableName, strings.Join(setClause, ", "), whereClause)
	return query, args, nil
}

func (s *StudioServer) handleExportData(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract table name from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/export/")
	if path == "" {
		http.Error(w, "Table name required", http.StatusBadRequest)
		return
	}

	// Validate table name to prevent SQL injection
	if !isValidTableName(path) {
		http.Error(w, "Invalid table name", http.StatusBadRequest)
		return
	}

	// Get format from query parameters
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv" // default format
	}

	// Validate format
	if format != "csv" && format != "json" && format != "sql" {
		http.Error(w, "Invalid format. Supported formats: csv, json, sql", http.StatusBadRequest)
		return
	}

	// Get database pool
	pool, err := s.getPool()
	if err != nil {
		http.Error(w, "Database connection failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := context.Background()

	// Get all data from the table
	query := "SELECT * FROM \"" + path + "\" ORDER BY 1"
	rows, err := pool.Query(ctx, query)
	if err != nil {
		http.Error(w, "Failed to query table data: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Get column names
	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columns[i] = string(fd.Name)
	}

	// Scan data
	var data []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if scanErr := rows.Scan(valuePtrs...); scanErr != nil {
			http.Error(w, "Failed to scan row: "+scanErr.Error(), http.StatusInternalServerError)
			return
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if val == nil {
				row[col] = nil
			} else {
				row[col] = val
			}
		}
		data = append(data, row)
	}

	// Generate export content based on format
	var content string
	var filename string
	var contentType string

	switch format {
	case "csv":
		content, err = s.generateCSV(columns, data)
		filename = path + ".csv"
		contentType = "text/csv"
	case "json":
		content, err = s.generateJSON(data)
		filename = path + ".json"
		contentType = "application/json"
	case "sql":
		content, err = s.generateSQL(path, columns, data)
		filename = path + ".sql"
		contentType = "text/plain"
	}

	if err != nil {
		http.Error(w, "Failed to generate export: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set headers for file download
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))

	// Write content
	w.Write([]byte(content))
}

func (s *StudioServer) generateCSV(columns []string, data []map[string]interface{}) (string, error) {
	var result strings.Builder
	
	// Write header
	for i, col := range columns {
		if i > 0 {
			result.WriteString(",")
		}
		result.WriteString(s.escapeCSVField(col))
	}
	result.WriteString("\n")
	
	// Write data rows
	for _, row := range data {
		for i, col := range columns {
			if i > 0 {
				result.WriteString(",")
			}
			value := row[col]
			if value == nil {
				result.WriteString("")
			} else {
				result.WriteString(s.escapeCSVField(fmt.Sprint(value)))
			}
		}
		result.WriteString("\n")
	}
	
	return result.String(), nil
}

func (s *StudioServer) escapeCSVField(field string) string {
	// If field contains comma, quote, or newline, wrap in quotes and escape internal quotes
	if strings.ContainsAny(field, ",\"\n\r") {
		escaped := strings.ReplaceAll(field, "\"", "\"\"")
		return "\"" + escaped + "\""
	}
	return field
}

func (s *StudioServer) generateJSON(data []map[string]interface{}) (string, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(jsonData), nil
}

func (s *StudioServer) generateSQL(tableName string, columns []string, data []map[string]interface{}) (string, error) {
	var result strings.Builder
	
	// Write header comment
	result.WriteString("-- Export of table " + tableName + "\n")
	result.WriteString("-- Generated by Migrato Studio\n\n")
	
	// Write INSERT statements
	for _, row := range data {
		result.WriteString("INSERT INTO " + tableName + " (")
		
		// Write column names
		for i, col := range columns {
			if i > 0 {
				result.WriteString(", ")
			}
			result.WriteString("\"" + col + "\"")
		}
		result.WriteString(") VALUES (")
		
		// Write values
		for i, col := range columns {
			if i > 0 {
				result.WriteString(", ")
			}
			value := row[col]
			if value == nil {
				result.WriteString("NULL")
			} else {
				switch v := value.(type) {
				case string:
					result.WriteString("'" + s.escapeSQLString(v) + "'")
				case int, int32, int64:
					result.WriteString(fmt.Sprint(v))
				case float32, float64:
					result.WriteString(fmt.Sprint(v))
				case bool:
					if v {
						result.WriteString("true")
					} else {
						result.WriteString("false")
					}
				default:
					result.WriteString("'" + fmt.Sprint(v) + "'")
				}
			}
		}
		result.WriteString(");\n")
	}
	
	return result.String(), nil
}

func (s *StudioServer) escapeSQLString(str string) string {
	return strings.ReplaceAll(str, "'", "''")
}

func (s *StudioServer) handleImportData(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 32MB)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get table name from form
	tableName := r.FormValue("table")
	if tableName == "" {
		http.Error(w, "Table name required", http.StatusBadRequest)
		return
	}

	// Validate table name to prevent SQL injection
	if !isValidTableName(tableName) {
		http.Error(w, "Invalid table name", http.StatusBadRequest)
		return
	}

	// Get format from form
	format := r.FormValue("format")
	if format == "" {
		format = "csv" // default format
	}

	// Validate format
	if format != "csv" && format != "json" && format != "sql" {
		http.Error(w, "Invalid format. Supported formats: csv, json, sql", http.StatusBadRequest)
		return
	}

	// Get uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get uploaded file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get database pool
	pool, err := s.getPool()
	if err != nil {
		http.Error(w, "Database connection failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := context.Background()

	// Parse and import data based on format
	var importResult map[string]interface{}
	switch format {
	case "csv":
		importResult, err = s.importCSV(ctx, pool, tableName, string(content))
	case "json":
		importResult, err = s.importJSON(ctx, pool, tableName, content)
	case "sql":
		importResult, err = s.importSQL(ctx, pool, string(content))
	}

	if err != nil {
		http.Error(w, "Import failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"success": true,
		"message": "Data imported successfully",
		"filename": header.Filename,
		"format": format,
		"table": tableName,
		"result": importResult,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *StudioServer) importCSV(ctx context.Context, pool *pgxpool.Pool, tableName, content string) (map[string]interface{}, error) {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("CSV file must have at least a header and one data row")
	}

	// Parse header
	header := s.parseCSVLine(lines[0])
	if len(header) == 0 {
		return nil, fmt.Errorf("invalid CSV header")
	}

	// Validate columns exist in table
	if err := s.validateColumns(ctx, pool, tableName, header); err != nil {
		return nil, err
	}

	// Begin transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Prepare INSERT statement
	placeholders := make([]string, len(header))
	for i := range placeholders {
		placeholders[i] = "$" + strconv.Itoa(i+1)
	}
	
	query := "INSERT INTO " + tableName + " (" + strings.Join(header, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ")"

	// Insert data rows
	insertedRows := 0
	for i, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}

		values := s.parseCSVLine(line)
		if len(values) != len(header) {
			return nil, fmt.Errorf("row %d has %d values, expected %d", i+2, len(values), len(header))
		}

		// Convert values to interface slice
		args := make([]interface{}, len(values))
		for j, val := range values {
			if val == "" {
				args[j] = nil
			} else {
				args[j] = val
			}
		}

		_, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to insert row %d: %w", i+2, err)
		}
		insertedRows++
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return map[string]interface{}{
		"inserted_rows": insertedRows,
		"total_rows":    len(lines) - 1,
	}, nil
}

func (s *StudioServer) parseCSVLine(line string) []string {
	var result []string
	var current strings.Builder
	inQuotes := false
	
	for i := 0; i < len(line); i++ {
		char := line[i]
		
		if char == '"' {
			if inQuotes && i+1 < len(line) && line[i+1] == '"' {
				// Escaped quote
				current.WriteByte('"')
				i++ // Skip next quote
			} else {
				// Toggle quote state
				inQuotes = !inQuotes
			}
		} else if char == ',' && !inQuotes {
			// End of field
			result = append(result, current.String())
			current.Reset()
		} else {
			current.WriteByte(char)
		}
	}
	
	// Add last field
	result = append(result, current.String())
	return result
}

func (s *StudioServer) importJSON(ctx context.Context, pool *pgxpool.Pool, tableName string, content []byte) (map[string]interface{}, error) {
	var data []map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("JSON file contains no data")
	}

	// Get column names from first row
	columns := make([]string, 0, len(data[0]))
	for col := range data[0] {
		columns = append(columns, col)
	}

	// Validate columns exist in table
	if err := s.validateColumns(ctx, pool, tableName, columns); err != nil {
		return nil, err
	}

	// Begin transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Prepare INSERT statement
	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "$" + strconv.Itoa(i+1)
	}
	
	query := "INSERT INTO " + tableName + " (" + strings.Join(columns, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ")"

	// Insert data rows
	insertedRows := 0
	for i, row := range data {
		args := make([]interface{}, len(columns))
		for j, col := range columns {
			args[j] = row[col]
		}

		_, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to insert row %d: %w", i+1, err)
		}
		insertedRows++
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return map[string]interface{}{
		"inserted_rows": insertedRows,
		"total_rows":    len(data),
	}, nil
}

func (s *StudioServer) importSQL(ctx context.Context, pool *pgxpool.Pool, content string) (map[string]interface{}, error) {
	// Split content into individual statements
	statements := strings.Split(content, ";")
	
	// Begin transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	executedStatements := 0
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}

		_, err := tx.Exec(ctx, stmt)
		if err != nil {
			return nil, fmt.Errorf("failed to execute statement: %w", err)
		}
		executedStatements++
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return map[string]interface{}{
		"executed_statements": executedStatements,
	}, nil
}

func (s *StudioServer) validateColumns(ctx context.Context, pool *pgxpool.Pool, tableName string, columns []string) error {
	query := `
		SELECT column_name 
		FROM information_schema.columns 
		WHERE table_name = $1 AND table_schema = 'public'
		ORDER BY ordinal_position`

	rows, err := pool.Query(ctx, query, tableName)
	if err != nil {
		return fmt.Errorf("failed to get table columns: %w", err)
	}
	defer rows.Close()

	existingColumns := make(map[string]bool)
	for rows.Next() {
		var colName pgtype.Text
		if err := rows.Scan(&colName); err != nil {
			return fmt.Errorf("failed to scan column name: %w", err)
		}
		existingColumns[colName.String] = true
	}

	// Check if all import columns exist in table
	var missingColumns []string
	for _, col := range columns {
		if !existingColumns[col] {
			missingColumns = append(missingColumns, col)
		}
	}

	if len(missingColumns) > 0 {
		return fmt.Errorf("columns not found in table %s: %s", tableName, strings.Join(missingColumns, ", "))
	}

	return nil
}