package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

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
	http.HandleFunc("/api/table/", server.handleTableData)
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
            <div class="text-2xl">🐘</div>
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
            <div id="tableView" class="h-full flex flex-col">
                <div class="flex-1 flex items-center justify-center">
                    <div class="text-center">
                        <div class="text-6xl mb-6 opacity-50">📊</div>
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
		w.Write([]byte(
			"// Migrato Studio - Database Browser\n" +
			"class MigratoStudio {\n" +
			"    constructor() {\n" +
			"        this.currentTable = null;\n" +
			"        this.currentPage = 1;\n" +
			"        this.pageSize = 50;\n" +
			"        this.searchTerm = '';\n" +
			"        this.loading = false;\n" +
			"        this.sidebarCollapsed = false;\n" +
			"        this.isDarkMode = true;\n" +
			"        this.init();\n" +
			"    }\n" +
			"    async init() {\n" +
			"        this.loadTheme();\n" +
			"        await this.loadTables();\n" +
			"        this.setupEventListeners();\n" +
			"    }\n" +
			"    async loadTables() {\n" +
			"        try {\n" +
			"            const response = await fetch(\"/api/tables\");\n" +
			"            const data = await response.json();\n" +
			"            this.renderTableList(data.tables);\n" +
			"        } catch (error) {\n" +
			"            console.error(\"Error loading tables:\", error);\n" +
			"            this.showError(\"Failed to load tables\");\n" +
			"        }\n" +
			"    }\n" +
			"    renderTableList(tables) {\n" +
			"        const tableList = document.getElementById(\"tableList\");\n" +
			"        tableList.innerHTML = '';\n" +
			"        \n" +
			"        if (tables.length === 0) {\n" +
			"            tableList.innerHTML = '<div class=\\\"text-slate-400 italic text-sm p-4\\\">No tables found</div>';\n" +
			"            return;\n" +
			"        }\n" +
			"        \n" +
			"        tables.forEach((table, index) => {\n" +
			"            const item = document.createElement(\"div\");\n" +
			"            item.className = 'group flex items-center space-x-3 px-4 py-3 rounded-lg cursor-pointer hover:bg-slate-700 transition-all duration-200 text-sm font-medium text-slate-300 hover:text-white slide-in';\n" +
			"            item.style.animationDelay = (index * 50) + 'ms';\n" +
			"            \n" +
			"            const icon = document.createElement(\"div\");\n" +
			"            icon.className = 'w-5 h-5 text-slate-500 group-hover:text-blue-400 transition-colors';\n" +
			"            icon.innerHTML = '📋';\n" +
			"            \n" +
			"            const text = document.createElement(\"span\");\n" +
			"            text.textContent = table;\n" +
			"            \n" +
			"            item.appendChild(icon);\n" +
			"            item.appendChild(text);\n" +
			"            item.onclick = () => this.selectTable(table);\n" +
			"            tableList.appendChild(item);\n" +
			"        });\n" +
			"    }\n" +
			"    async selectTable(tableName) {\n" +
			"        this.currentTable = tableName;\n" +
			"        this.currentPage = 1;\n" +
			"        \n" +
			"        // Update active state\n" +
			"        document.querySelectorAll(\"[onclick*='selectTable']\").forEach(item => {\n" +
			"            item.classList.remove(\"bg-blue-600\", \"text-white\");\n" +
			"            item.classList.add(\"text-slate-300\", \"hover:bg-slate-700\");\n" +
			"            item.querySelector('div').classList.remove(\"text-blue-400\");\n" +
			"            item.querySelector('div').classList.add(\"text-slate-500\");\n" +
			"        });\n" +
			"        \n" +
			"        event.target.closest('div').classList.remove(\"text-slate-300\", \"hover:bg-slate-700\");\n" +
			"        event.target.closest('div').classList.add(\"bg-blue-600\", \"text-white\");\n" +
			"        event.target.closest('div').querySelector('div').classList.remove(\"text-slate-500\");\n" +
			"        event.target.closest('div').querySelector('div').classList.add(\"text-blue-400\");\n" +
			"        \n" +
			"        await this.loadTableData();\n" +
			"    }\n" +
			"    async loadTableData() {\n" +
			"        if (!this.currentTable || this.loading) return;\n" +
			"        \n" +
			"        this.loading = true;\n" +
			"        this.showLoading();\n" +
			"        \n" +
			"        const params = new URLSearchParams({\n" +
			"            page: this.currentPage,\n" +
			"            limit: this.pageSize,\n" +
			"            search: this.searchTerm\n" +
			"        });\n" +
			"        \n" +
			"        try {\n" +
			"            const response = await fetch(\"/api/table/\" + this.currentTable + \"?\" + params);\n" +
			"            const data = await response.json();\n" +
			"            this.renderTableData(data);\n" +
			"        } catch (error) {\n" +
			"            console.error(\"Error loading table data:\", error);\n" +
			"            this.showError(\"Failed to load table data\");\n" +
			"        } finally {\n" +
			"            this.loading = false;\n" +
			"        }\n" +
			"    }\n" +
			"    showLoading() {\n" +
			"        const tableView = document.getElementById(\"tableView\");\n" +
			"        tableView.innerHTML = '<div class=\\\"flex-1 flex items-center justify-center\\\"><div class=\\\"text-center\\\"><div class=\\\"animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mx-auto mb-4\\\"></div><div class=\\\"text-slate-400\\\">Loading data...</div></div></div>';\n" +
			"    }\n" +
			"    renderTableData(data) {\n" +
			"        const tableView = document.getElementById(\"tableView\");\n" +
			"        \n" +
			"        if (!data.data || data.data.length === 0) {\n" +
			"            tableView.innerHTML = '<div class=\\\"flex-1 flex items-center justify-center\\\"><div class=\\\"text-center\\\"><div class=\\\"text-4xl mb-4 opacity-50\\\">📭</div><div class=\\\"text-slate-400 text-lg\\\">No data found</div></div></div>';\n" +
			"            return;\n" +
			"        }\n" +
			"        \n" +
			"        const controls = this.createTableControls();\n" +
			"        const table = this.createDataTable(data.data);\n" +
			"        const pagination = this.createPagination(data);\n" +
			"        \n" +
			"        tableView.innerHTML = '';\n" +
			"        tableView.className = 'h-full flex flex-col p-6';\n" +
			"        tableView.appendChild(controls);\n" +
			"        tableView.appendChild(table);\n" +
			"        tableView.appendChild(pagination);\n" +
			"        \n" +
			"        // Add fade-in animation\n" +
			"        tableView.classList.add('fade-in');\n" +
			"    }\n" +
			"    createTableControls() {\n" +
			"        const controls = document.createElement(\"div\");\n" +
			"        controls.className = 'flex items-center justify-between mb-6';\n" +
			"        \n" +
			"        const title = document.createElement(\"h2\");\n" +
			"        title.className = 'text-2xl font-bold text-white';\n" +
			"        title.textContent = this.currentTable;\n" +
			"        \n" +
			"        const searchBox = document.createElement(\"div\");\n" +
			"        searchBox.className = 'relative';\n" +
			"        searchBox.innerHTML = '<input type=\\\"text\\\" placeholder=\\\"Search in table...\\\" value=\\\"' + this.searchTerm + '\\\" class=\\\"pl-10 pr-4 py-3 bg-slate-800 border border-slate-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 w-80 text-white placeholder-slate-400\\\">';\n" +
			"        searchBox.innerHTML += '<svg class=\\\"absolute left-3 top-3.5 h-5 w-5 text-slate-400\\\" fill=\\\"none\\\" stroke=\\\"currentColor\\\" viewBox=\\\"0 0 24 24\\\"><path stroke-linecap=\\\"round\\\" stroke-linejoin=\\\"round\\\" stroke-width=\\\"2\\\" d=\\\"M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z\\\"></path></svg>';\n" +
			"        \n" +
			"        const input = searchBox.querySelector(\"input\");\n" +
			"        let debounceTimer;\n" +
			"        input.addEventListener(\"input\", (e) => {\n" +
			"            clearTimeout(debounceTimer);\n" +
			"            debounceTimer = setTimeout(() => {\n" +
			"                this.searchTerm = e.target.value;\n" +
			"                this.currentPage = 1;\n" +
			"                this.loadTableData();\n" +
			"            }, 300);\n" +
			"        });\n" +
			"        \n" +
			"        controls.appendChild(title);\n" +
			"        controls.appendChild(searchBox);\n" +
			"        return controls;\n" +
			"    }\n" +
			"    createDataTable(data) {\n" +
			"        const tableContainer = document.createElement(\"div\");\n" +
			"        tableContainer.className = 'flex-1 overflow-hidden bg-slate-800 rounded-lg border border-slate-700';\n" +
			"        \n" +
			"        const tableWrapper = document.createElement(\"div\");\n" +
			"        tableWrapper.className = 'h-full overflow-auto';\n" +
			"        \n" +
			"        const table = document.createElement(\"table\");\n" +
			"        table.className = 'min-w-full divide-y divide-slate-700';\n" +
			"        \n" +
			"        const thead = document.createElement(\"thead\");\n" +
			"        thead.className = 'bg-slate-700 sticky top-0';\n" +
			"        const headerRow = document.createElement(\"tr\");\n" +
			"        \n" +
			"        Object.keys(data[0]).forEach(key => {\n" +
			"            const th = document.createElement(\"th\");\n" +
			"            th.className = 'px-6 py-4 text-left text-xs font-semibold text-slate-300 uppercase tracking-wider';\n" +
			"            th.textContent = key;\n" +
			"            headerRow.appendChild(th);\n" +
			"        });\n" +
			"        \n" +
			"        thead.appendChild(headerRow);\n" +
			"        table.appendChild(thead);\n" +
			"        \n" +
			"        const tbody = document.createElement(\"tbody\");\n" +
			"        tbody.className = 'bg-slate-800 divide-y divide-slate-700';\n" +
			"        \n" +
			"        data.forEach((row, index) => {\n" +
			"            const tr = document.createElement(\"tr\");\n" +
			"            tr.className = index % 2 === 0 ? 'bg-slate-800' : 'bg-slate-750';\n" +
			"            tr.className += ' hover:bg-slate-700 transition-colors duration-150';\n" +
			"            \n" +
			"            Object.values(row).forEach(value => {\n" +
			"                const td = document.createElement(\"td\");\n" +
			"                td.className = 'px-6 py-4 whitespace-nowrap text-sm text-slate-300';\n" +
			"                \n" +
			"                if (value === null) {\n" +
			"                    td.innerHTML = '<span class=\\\"text-slate-500 italic\\\">NULL</span>';\n" +
			"                } else {\n" +
			"                    td.textContent = String(value);\n" +
			"                }\n" +
			"                \n" +
			"                tr.appendChild(td);\n" +
			"            });\n" +
			"            tbody.appendChild(tr);\n" +
			"        });\n" +
			"        \n" +
			"        table.appendChild(tbody);\n" +
			"        tableWrapper.appendChild(table);\n" +
			"        tableContainer.appendChild(tableWrapper);\n" +
			"        return tableContainer;\n" +
			"    }\n" +
			"    createPagination(data) {\n" +
			"        const pagination = document.createElement(\"div\");\n" +
			"        pagination.className = 'flex items-center justify-between mt-6 pt-6 border-t border-slate-700';\n" +
			"        \n" +
			"        const totalPages = Math.ceil(data.total / this.pageSize);\n" +
			"        \n" +
			"        const info = document.createElement(\"div\");\n" +
			"        info.className = 'text-sm text-slate-400';\n" +
			"        info.textContent = 'Showing ' + ((this.currentPage - 1) * this.pageSize + 1) + ' to ' + Math.min(this.currentPage * this.pageSize, data.total) + ' of ' + data.total + ' rows';\n" +
			"        \n" +
			"        const controls = document.createElement(\"div\");\n" +
			"        controls.className = 'flex space-x-2';\n" +
			"        \n" +
			"        const prevBtn = document.createElement(\"button\");\n" +
			"        prevBtn.textContent = '← Previous';\n" +
			"        prevBtn.disabled = this.currentPage <= 1;\n" +
			"        prevBtn.className = 'px-4 py-2 text-sm font-medium text-slate-300 bg-slate-800 border border-slate-600 rounded-lg hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors';\n" +
			"        prevBtn.onclick = () => {\n" +
			"            if (this.currentPage > 1) {\n" +
			"                this.currentPage--;\n" +
			"                this.loadTableData();\n" +
			"            }\n" +
			"        };\n" +
			"        \n" +
			"        const nextBtn = document.createElement(\"button\");\n" +
			"        nextBtn.textContent = 'Next →';\n" +
			"        nextBtn.disabled = this.currentPage >= totalPages;\n" +
			"        nextBtn.className = 'px-4 py-2 text-sm font-medium text-slate-300 bg-slate-800 border border-slate-600 rounded-lg hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors';\n" +
			"        nextBtn.onclick = () => {\n" +
			"            if (this.currentPage < totalPages) {\n" +
			"                this.currentPage++;\n" +
			"                this.loadTableData();\n" +
			"            }\n" +
			"        };\n" +
			"        \n" +
			"        controls.appendChild(prevBtn);\n" +
			"        controls.appendChild(nextBtn);\n" +
			"        \n" +
			"        pagination.appendChild(info);\n" +
			"        pagination.appendChild(controls);\n" +
			"        return pagination;\n" +
			"    }\n" +
			"    setupEventListeners() {\n" +
			"        // Sidebar toggle functionality\n" +
			"        const sidebarToggle = document.getElementById('sidebarToggle');\n" +
			"        const sidebarClose = document.getElementById('sidebarClose');\n" +
			"        const sidebar = document.getElementById('sidebar');\n" +
			"        const sidebarOverlay = document.getElementById('sidebarOverlay');\n" +
			"        \n" +
			"        // Toggle sidebar on button click\n" +
			"        sidebarToggle.addEventListener('click', () => {\n" +
			"            this.toggleSidebar();\n" +
			"        });\n" +
			"        \n" +
			"        // Close sidebar on close button (mobile)\n" +
			"        sidebarClose.addEventListener('click', () => {\n" +
			"            this.closeSidebar();\n" +
			"        });\n" +
			"        \n" +
			"        // Close sidebar on overlay click (mobile)\n" +
			"        sidebarOverlay.addEventListener('click', () => {\n" +
			"            this.closeSidebar();\n" +
			"        });\n" +
			"        \n" +
			"        // Theme toggle functionality\n" +
			"        const themeToggle = document.getElementById('themeToggle');\n" +
			"        const themeIcon = document.getElementById('themeIcon');\n" +
			"        \n" +
			"        // Toggle theme on button click\n" +
			"        themeToggle.addEventListener('click', () => {\n" +
			"            this.toggleTheme();\n" +
			"        });\n" +
			"        \n" +
			"        // Add keyboard shortcuts\n" +
			"        document.addEventListener('keydown', (e) => {\n" +
			"            if (e.ctrlKey || e.metaKey) {\n" +
			"                switch(e.key) {\n" +
			"                    case 'f':\n" +
			"                        e.preventDefault();\n" +
			"                        const searchInput = document.querySelector('input[placeholder*=\"Search\"]');\n" +
			"                        if (searchInput) searchInput.focus();\n" +
			"                        break;\n" +
								"                    case 'b':\n" +
					"                        e.preventDefault();\n" +
					"                        this.toggleSidebar();\n" +
					"                        break;\n" +
					"                    case 't':\n" +
					"                        e.preventDefault();\n" +
					"                        this.toggleTheme();\n" +
					"                        break;\n" +
			"                }\n" +
			"            }\n" +
			"        });\n" +
			"    }\n" +
			"    toggleSidebar() {\n" +
			"        const sidebar = document.getElementById('sidebar');\n" +
			"        const sidebarOverlay = document.getElementById('sidebarOverlay');\n" +
			"        \n" +
			"        if (this.sidebarCollapsed) {\n" +
			"            // Expand sidebar\n" +
			"            sidebar.classList.remove('sidebar-collapsed');\n" +
			"            sidebarOverlay.classList.remove('active');\n" +
			"            this.sidebarCollapsed = false;\n" +
			"        } else {\n" +
			"            // Collapse sidebar\n" +
			"            sidebar.classList.add('sidebar-collapsed');\n" +
			"            if (window.innerWidth < 1024) {\n" +
			"                sidebarOverlay.classList.add('active');\n" +
			"            }\n" +
			"            this.sidebarCollapsed = true;\n" +
			"        }\n" +
			"    }\n" +
			"    closeSidebar() {\n" +
			"        const sidebar = document.getElementById('sidebar');\n" +
			"        const sidebarOverlay = document.getElementById('sidebarOverlay');\n" +
			"        \n" +
			"        sidebar.classList.remove('sidebar-open');\n" +
			"        sidebarOverlay.classList.remove('active');\n" +
			"    }\n" +
			"    toggleTheme() {\n" +
			"        const body = document.body;\n" +
			"        const themeIcon = document.getElementById('themeIcon');\n" +
			"        \n" +
			"        if (this.isDarkMode) {\n" +
			"            // Switch to light mode\n" +
			"            body.classList.add('light-mode');\n" +
			"            themeIcon.innerHTML = '<path stroke-linecap=\\\"round\\\" stroke-linejoin=\\\"round\\\" stroke-width=\\\"2\\\" d=\\\"M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z\\\"></path>';\n" +
			"            this.isDarkMode = false;\n" +
			"        } else {\n" +
			"            // Switch to dark mode\n" +
			"            body.classList.remove('light-mode');\n" +
			"            themeIcon.innerHTML = '<path stroke-linecap=\\\"round\\\" stroke-linejoin=\\\"round\\\" stroke-width=\\\"2\\\" d=\\\"M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z\\\"></path>';\n" +
			"            this.isDarkMode = true;\n" +
			"        }\n" +
			"        \n" +
			"        // Save theme preference\n" +
			"        localStorage.setItem('migrato-theme', this.isDarkMode ? 'dark' : 'light');\n" +
			"    }\n" +
			"    loadTheme() {\n" +
			"        const savedTheme = localStorage.getItem('migrato-theme');\n" +
			"        const themeIcon = document.getElementById('themeIcon');\n" +
			"        \n" +
			"        if (savedTheme === 'light') {\n" +
			"            document.body.classList.add('light-mode');\n" +
			"            themeIcon.innerHTML = '<path stroke-linecap=\\\"round\\\" stroke-linejoin=\\\"round\\\" stroke-width=\\\"2\\\" d=\\\"M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z\\\"></path>';\n" +
			"            this.isDarkMode = false;\n" +
			"        }\n" +
			"    }\n" +
			"    showError(message) {\n" +
			"        const tableView = document.getElementById(\"tableView\");\n" +
			"        tableView.innerHTML = '<div class=\\\"flex-1 flex items-center justify-center\\\"><div class=\\\"text-center\\\"><div class=\\\"text-4xl mb-4\\\">⚠️</div><div class=\\\"text-red-400 font-medium text-lg\\\">' + message + '</div></div></div>';\n" +
			"    }\n" +
			"}\n" +
			"document.addEventListener(\"DOMContentLoaded\", () => {\n" +
			"    new MigratoStudio();\n" +
			"});\n",
		))
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