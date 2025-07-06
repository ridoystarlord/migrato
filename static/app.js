// Migrato Studio - Database Browser
class MigratoStudio {
  constructor() {
    this.currentTable = null;
    this.currentPage = 1;
    this.pageSize = 10; // Start with 10 rows
    this.searchTerm = "";
    this.loading = false;
    this.sidebarCollapsed = false;
    this.isDarkMode = true;
    this.editMode = false;
    this.selectedRows = [];
    this.init();
  }

  async init() {
    this.loadTheme();
    await this.loadTables();
    this.setupEventListeners();
    // Don't load relationships immediately - load them when the tab is viewed
  }

  async loadTables() {
    try {
      const response = await fetch("/api/tables");
      const data = await response.json();
      this.renderTableList(data.tables);
    } catch (error) {
      console.error("Error loading tables:", error);
      this.showError("Failed to load tables");
    }
  }

  renderTableList(tables) {
    const tableList = document.getElementById("tableList");
    tableList.innerHTML = "";

    if (tables.length === 0) {
      tableList.innerHTML =
        '<div class="text-slate-400 italic text-sm p-4">No tables found</div>';
      return;
    }

    tables.forEach((table, index) => {
      const item = document.createElement("div");
      item.className =
        "group flex items-center space-x-3 px-3 py-2.5 rounded-lg cursor-pointer hover:bg-slate-700 transition-all duration-200 text-sm font-medium text-slate-300 hover:text-white slide-in table-item";
      item.style.animationDelay = index * 30 + "ms";
      item.setAttribute("data-table", table);

      const icon = document.createElement("div");
      icon.className =
        "w-4 h-4 text-slate-500 group-hover:text-blue-400 transition-colors flex-shrink-0";
      icon.innerHTML = "📋";

      const text = document.createElement("span");
      text.textContent = table;
      text.className = "truncate";

      item.appendChild(icon);
      item.appendChild(text);
      item.onclick = () => this.selectTable(table);
      tableList.appendChild(item);
    });
  }

  async selectTable(tableName) {
    this.currentTable = tableName;
    this.currentPage = 1;

    // Clear all active states
    document.querySelectorAll(".table-item").forEach((item) => {
      item.classList.remove("bg-blue-600", "text-white");
      item.classList.add("text-slate-300", "hover:bg-slate-700");
      const icon = item.querySelector("div");
      if (icon) {
        icon.classList.remove("text-blue-400");
        icon.classList.add("text-slate-500");
      }
    });

    // Set active state for selected table
    const selectedItem = document.querySelector(`[data-table="${tableName}"]`);
    if (selectedItem) {
      selectedItem.classList.remove("text-slate-300", "hover:bg-slate-700");
      selectedItem.classList.add("bg-blue-600", "text-white");
      const icon = selectedItem.querySelector("div");
      if (icon) {
        icon.classList.remove("text-slate-500");
        icon.classList.add("text-blue-400");
      }
    }

    await this.loadTableData();
  }

  async loadTableData() {
    if (!this.currentTable || this.loading) return;

    this.loading = true;
    this.showLoading();

    const params = new URLSearchParams({
      page: this.currentPage,
      limit: this.pageSize,
      search: this.searchTerm,
    });

    try {
      const response = await fetch(
        "/api/table/" + this.currentTable + "?" + params
      );
      const data = await response.json();
      this.renderTableData(data);
    } catch (error) {
      console.error("Error loading table data:", error);
      this.showError("Failed to load table data");
    } finally {
      this.loading = false;
    }
  }

  showLoading() {
    const tableView = document.getElementById("tableView");
    tableView.innerHTML =
      '<div class="flex-1 flex items-center justify-center"><div class="text-center"><div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mx-auto mb-4"></div><div class="text-slate-400">Loading data...</div></div></div>';
  }

  renderTableData(data) {
    const tableView = document.getElementById("tableView");

    if (!data.data || data.data.length === 0) {
      tableView.innerHTML =
        '<div class="flex-1 flex items-center justify-center"><div class="text-center"><div class="text-4xl mb-4 opacity-50">EMPTY</div><div class="text-slate-400 text-lg">No data found</div></div></div>';
      return;
    }

    const controls = this.createTableControls();
    const table = this.createDataTable(data.data, data.columns);
    const pagination = this.createPagination(data);

    tableView.innerHTML = "";
    tableView.className = "h-full flex flex-col p-6";
    tableView.appendChild(controls);
    tableView.appendChild(table);
    tableView.appendChild(pagination);

    // Add fade-in animation
    tableView.classList.add("fade-in");

    // Enable inline editing if edit mode is active
    if (this.editMode) {
      this.enableInlineEditing();
    }
  }

  createTableControls() {
    const controls = document.createElement("div");
    controls.className = "flex items-center justify-between mb-6 gap-4";

    const title = document.createElement("h2");
    title.className = "text-2xl font-bold text-white";
    title.textContent = this.currentTable;

    // Search box (grow)
    const searchBox = document.createElement("input");
    searchBox.type = "text";
    searchBox.placeholder = "Search...";
    searchBox.className =
      "flex-grow px-4 py-2 rounded-lg bg-slate-700 text-white border border-slate-600 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 text-sm";
    searchBox.value = this.searchTerm;
    searchBox.addEventListener("input", (e) => {
      this.searchTerm = e.target.value;
      this.currentPage = 1;
      this.loadTableData();
    });

    // Edit toggle
    const editToggle = document.createElement("button");
    editToggle.className =
      "px-4 py-2 bg-slate-700 text-slate-300 rounded-lg hover:bg-blue-600 hover:text-white transition-colors text-sm font-medium" +
      (this.editMode ? " bg-blue-600 text-white" : "");
    editToggle.textContent = this.editMode ? "Editing" : "Edit";
    editToggle.onclick = () => this.toggleEditMode();

    // Import/Export dropdown (right aligned)
    const importExportDropdown = document.createElement("div");
    importExportDropdown.className = "relative";
    importExportDropdown.innerHTML =
      '<button class="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 focus:ring-2 focus:ring-green-500 transition-colors flex items-center">Import/Export</button>';
    const importExportMenu = document.createElement("div");
    importExportMenu.className =
      "absolute right-0 mt-2 w-48 bg-slate-800 border border-slate-600 rounded-lg shadow-lg z-50 hidden";
    importExportMenu.innerHTML =
      '<div class="py-1">' +
      '<a href="#" class="block px-4 py-2 text-sm text-slate-300 hover:bg-slate-700" data-format="csv">Export as CSV</a>' +
      '<a href="#" class="block px-4 py-2 text-sm text-slate-300 hover:bg-slate-700" data-format="json">Export as JSON</a>' +
      '<a href="#" class="block px-4 py-2 text-sm text-slate-300 hover:bg-slate-700" data-format="sql">Export as SQL</a>' +
      '<a href="#" class="block px-4 py-2 text-sm text-slate-300 hover:bg-slate-700" data-action="import">Import Data</a>' +
      "</div>";
    importExportDropdown.appendChild(importExportMenu);
    const importExportBtn = importExportDropdown.querySelector("button");
    importExportBtn.addEventListener("click", (e) => {
      e.stopPropagation();
      importExportMenu.classList.toggle("hidden");
    });
    importExportMenu.addEventListener("click", (e) => {
      e.preventDefault();
      const format = e.target.getAttribute("data-format");
      const action = e.target.getAttribute("data-action");
      if (format) {
        this.exportData(format);
        importExportMenu.classList.add("hidden");
      } else if (action === "import") {
        this.showImportModal();
        importExportMenu.classList.add("hidden");
      }
    });

    // Delete Selected button container
    const deleteBtnContainer = document.createElement("div");
    deleteBtnContainer.id = "deleteSelectedBtnContainer";
    deleteBtnContainer.className = "ml-2";

    // Controls layout
    controls.appendChild(title);
    controls.appendChild(searchBox);
    controls.appendChild(editToggle);
    controls.appendChild(importExportDropdown);
    controls.appendChild(deleteBtnContainer);
    // Save reference for later updates
    this.deleteBtnContainer = deleteBtnContainer;
    return controls;
  }

  createDataTable(data, columns) {
    if (!columns || columns.length === 0) {
      columns = data[0] ? Object.keys(data[0]) : [];
    }

    // Single table container
    const tableContainer = document.createElement("div");
    tableContainer.className =
      "table-outer bg-slate-800 rounded-lg border border-slate-700";

    // Single scrollable container
    const scrollDiv = document.createElement("div");
    scrollDiv.className = "table-scroll";

    // Single table with sticky header
    const table = document.createElement("table");
    table.className = "data-table";

    // Header
    const thead = document.createElement("thead");
    thead.className = "bg-slate-700";
    const headerRow = document.createElement("tr");

    // Checkbox column header
    const thCheckbox = document.createElement("th");
    thCheckbox.className = "checkbox-col border-b border-slate-600";
    const selectAll = document.createElement("input");
    selectAll.type = "checkbox";
    selectAll.addEventListener("change", (e) => {
      const checked = e.target.checked;
      this.selectedRows = checked ? data.map((row) => row[columns[0]]) : [];
      this.updateRowCheckboxes();
      this.updateDeleteButton();
    });
    thCheckbox.appendChild(selectAll);
    headerRow.appendChild(thCheckbox);

    // Data columns header
    columns.forEach((key, idx) => {
      const th = document.createElement("th");
      th.className =
        "px-3 py-2 text-left text-xs font-semibold text-slate-300 uppercase tracking-wider border-b border-slate-600" +
        (idx !== columns.length - 1 ? " border-r border-slate-700" : "");
      th.textContent = key;
      headerRow.appendChild(th);
    });

    thead.appendChild(headerRow);
    table.appendChild(thead);

    // Body
    const tbody = document.createElement("tbody");
    tbody.className = "bg-slate-800";

    data.forEach((row, index) => {
      const tr = document.createElement("tr");
      tr.className =
        (index % 2 === 0 ? "bg-slate-800" : "bg-slate-750") +
        " hover:bg-slate-700 transition-colors duration-150";

      // Checkbox cell
      const tdCheckbox = document.createElement("td");
      tdCheckbox.className = "checkbox-col border-b border-slate-700";
      const rowCheckbox = document.createElement("input");
      rowCheckbox.type = "checkbox";
      rowCheckbox.checked = this.selectedRows.includes(row[columns[0]]);
      rowCheckbox.addEventListener("change", (e) => {
        if (e.target.checked) {
          this.selectedRows.push(row[columns[0]]);
        } else {
          this.selectedRows = this.selectedRows.filter(
            (id) => id !== row[columns[0]]
          );
        }
        this.updateRowCheckboxes();
        this.updateDeleteButton();
      });
      tdCheckbox.appendChild(rowCheckbox);
      tr.appendChild(tdCheckbox);

      // Data cells
      const rowKeys = Object.keys(row);
      const firstKey = rowKeys[0];
      tr.setAttribute("data-row-id", firstKey);
      tr.setAttribute("data-row-id-value", String(row[firstKey]));

      columns.forEach((key, idx) => {
        const value = row[key];
        const td = document.createElement("td");
        td.className =
          "px-3 py-2 text-sm text-slate-300 border-b border-slate-700" +
          (idx !== columns.length - 1 ? " border-r border-slate-700" : "");
        td.setAttribute("data-column", key);
        td.setAttribute("data-editable", "true");

        if (value === null) {
          td.innerHTML =
            '<span class="text-slate-500 italic text-xs">NULL</span>';
        } else {
          const valueStr = String(value);
          if (valueStr.length > 50) {
            td.innerHTML = `<span title="${valueStr}">${valueStr.substring(
              0,
              50
            )}...</span>`;
          } else {
            td.textContent = valueStr;
          }
        }
        tr.appendChild(td);
      });
      tbody.appendChild(tr);
    });

    table.appendChild(tbody);
    scrollDiv.appendChild(table);
    tableContainer.appendChild(scrollDiv);

    // Update functions
    this.updateDeleteButton = () => {
      if (!this.deleteBtnContainer) return;
      this.deleteBtnContainer.innerHTML = "";
      if (this.selectedRows.length > 0) {
        const btn = document.createElement("button");
        btn.id = "deleteSelectedBtn";
        btn.className =
          "ml-2 px-6 py-2 bg-red-600 text-white rounded-lg shadow hover:bg-red-700 transition-colors font-semibold text-base";
        btn.textContent = `Delete Selected (${this.selectedRows.length})`;
        btn.onclick = () => this.deleteSelectedRows();
        this.deleteBtnContainer.appendChild(btn);
      }
    };

    this.updateRowCheckboxes = () => {
      // Update all row checkboxes to match selectedRows
      document
        .querySelectorAll("tbody input[type='checkbox']")
        .forEach((cb, idx) => {
          const row = data[idx];
          if (!row) return;
          cb.checked = this.selectedRows.includes(row[columns[0]]);
        });

      // Update select-all checkbox
      const selectAll = document.querySelector("thead input[type='checkbox']");
      if (selectAll) {
        selectAll.checked =
          this.selectedRows.length === data.length && data.length > 0;
        selectAll.indeterminate =
          this.selectedRows.length > 0 &&
          this.selectedRows.length < data.length;
      }
    };

    setTimeout(() => {
      this.updateRowCheckboxes();
      this.updateDeleteButton();
    }, 0);

    return tableContainer;
  }

  createPagination(data) {
    const pagination = document.createElement("div");
    pagination.className =
      "flex-shrink-0 flex flex-col sm:flex-row items-center justify-between gap-4 pt-4 border-t border-slate-700 bg-slate-800 p-4 rounded-b-lg";

    const totalPages = Math.ceil(data.total / this.pageSize);

    const info = document.createElement("div");
    info.className = "text-sm text-slate-400 text-center sm:text-left";

    if (data.total === 0) {
      info.textContent = "No data found";
    } else {
      const start = (this.currentPage - 1) * this.pageSize + 1;
      const end = Math.min(this.currentPage * this.pageSize, data.total);
      info.textContent = `Showing ${start} to ${end} of ${data.total} rows (Page ${this.currentPage} of ${totalPages})`;
    }

    const controls = document.createElement("div");
    controls.className = "flex items-center space-x-3 flex-wrap justify-center";

    // First page button
    const firstBtn = document.createElement("button");
    firstBtn.innerHTML = "⏮ First";
    firstBtn.disabled = this.currentPage <= 1;
    firstBtn.className =
      "px-3 py-2 text-sm font-medium text-slate-300 bg-slate-800 border border-slate-600 rounded-lg hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors";
    firstBtn.onclick = () => {
      if (this.currentPage > 1) {
        this.currentPage = 1;
        this.loadTableData();
      }
    };

    // Previous button
    const prevBtn = document.createElement("button");
    prevBtn.innerHTML = "← Previous";
    prevBtn.disabled = this.currentPage <= 1;
    prevBtn.className =
      "px-4 py-2 text-sm font-medium text-slate-300 bg-slate-800 border border-slate-600 rounded-lg hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors";
    prevBtn.onclick = () => {
      if (this.currentPage > 1) {
        this.currentPage--;
        this.loadTableData();
      }
    };

    // Page indicator
    const pageIndicator = document.createElement("span");
    pageIndicator.className =
      "px-4 py-2 text-sm font-medium text-slate-300 bg-slate-700 border border-slate-600 rounded-lg";
    pageIndicator.textContent = `${this.currentPage} / ${totalPages}`;

    // Next button
    const nextBtn = document.createElement("button");
    nextBtn.innerHTML = "Next →";
    nextBtn.disabled = this.currentPage >= totalPages;
    nextBtn.className =
      "px-4 py-2 text-sm font-medium text-slate-300 bg-slate-800 border border-slate-600 rounded-lg hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors";
    nextBtn.onclick = () => {
      if (this.currentPage < totalPages) {
        this.currentPage++;
        this.loadTableData();
      }
    };

    // Last page button
    const lastBtn = document.createElement("button");
    lastBtn.innerHTML = "Last ⏭";
    lastBtn.disabled = this.currentPage >= totalPages;
    lastBtn.className =
      "px-3 py-2 text-sm font-medium text-slate-300 bg-slate-800 border border-slate-600 rounded-lg hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors";
    lastBtn.onclick = () => {
      if (this.currentPage < totalPages) {
        this.currentPage = totalPages;
        this.loadTableData();
      }
    };

    // Page size selector
    const pageSizeContainer = document.createElement("div");
    pageSizeContainer.className = "flex items-center space-x-2 ml-4";

    const pageSizeLabel = document.createElement("label");
    pageSizeLabel.className = "text-sm text-slate-300";
    pageSizeLabel.textContent = "Rows:";

    const pageSizeSelect = document.createElement("select");
    pageSizeSelect.className =
      "px-2 py-1 bg-slate-700 border border-slate-600 rounded text-white text-sm focus:ring-2 focus:ring-blue-500";
    pageSizeSelect.innerHTML = `
      <option value="10" ${this.pageSize === 10 ? "selected" : ""}>10</option>
      <option value="25" ${this.pageSize === 25 ? "selected" : ""}>25</option>
      <option value="50" ${this.pageSize === 50 ? "selected" : ""}>50</option>
      <option value="100" ${
        this.pageSize === 100 ? "selected" : ""
      }>100</option>
    `;

    pageSizeSelect.addEventListener("change", (e) => {
      this.pageSize = parseInt(e.target.value);
      this.currentPage = 1; // Reset to first page
      this.loadTableData();
    });

    pageSizeContainer.appendChild(pageSizeLabel);
    pageSizeContainer.appendChild(pageSizeSelect);

    controls.appendChild(firstBtn);
    controls.appendChild(prevBtn);
    controls.appendChild(pageIndicator);
    controls.appendChild(nextBtn);
    controls.appendChild(lastBtn);
    controls.appendChild(pageSizeContainer);

    pagination.appendChild(info);
    pagination.appendChild(controls);
    return pagination;
  }

  setupEventListeners() {
    // Sidebar toggle functionality
    const sidebarToggle = document.getElementById("sidebarToggle");
    const sidebarClose = document.getElementById("sidebarClose");
    const sidebar = document.getElementById("sidebar");
    const sidebarOverlay = document.getElementById("sidebarOverlay");

    // Toggle sidebar on button click
    sidebarToggle.addEventListener("click", () => {
      this.toggleSidebar();
    });

    // Close sidebar on close button (mobile)
    sidebarClose.addEventListener("click", () => {
      this.closeSidebar();
    });

    // Close sidebar on overlay click (mobile)
    sidebarOverlay.addEventListener("click", () => {
      this.closeSidebar();
    });

    // Theme toggle functionality
    const themeToggle = document.getElementById("themeToggle");
    const themeIcon = document.getElementById("themeIcon");

    // Toggle theme on button click
    themeToggle.addEventListener("click", () => {
      this.toggleTheme();
    });

    // Add keyboard shortcuts
    document.addEventListener("keydown", (e) => {
      if (e.ctrlKey || e.metaKey) {
        switch (e.key) {
          case "f":
            e.preventDefault();
            const searchInput = document.querySelector(
              'input[placeholder*="Search"]'
            );
            if (searchInput) searchInput.focus();
            break;
          case "b":
            e.preventDefault();
            this.toggleSidebar();
            break;
          case "t":
            e.preventDefault();
            this.toggleTheme();
            break;
        }
      }
    });

    // Close dropdowns when clicking outside
    document.addEventListener("click", (e) => {
      const dropdowns = document.querySelectorAll(".absolute");
      dropdowns.forEach((dropdown) => {
        if (
          !dropdown.contains(e.target) &&
          !dropdown.classList.contains("hidden")
        ) {
          dropdown.classList.add("hidden");
        }
      });
    });
  }

  toggleSidebar() {
    const sidebar = document.getElementById("sidebar");
    const sidebarOverlay = document.getElementById("sidebarOverlay");

    if (this.sidebarCollapsed) {
      // Expand sidebar
      sidebar.classList.remove("sidebar-collapsed");
      sidebarOverlay.classList.remove("active");
      this.sidebarCollapsed = false;
    } else {
      // Collapse sidebar
      sidebar.classList.add("sidebar-collapsed");
      if (window.innerWidth < 1024) {
        sidebarOverlay.classList.add("active");
      }
      this.sidebarCollapsed = true;
    }
  }

  closeSidebar() {
    const sidebar = document.getElementById("sidebar");
    const sidebarOverlay = document.getElementById("sidebarOverlay");

    sidebar.classList.remove("sidebar-open");
    sidebarOverlay.classList.remove("active");
  }

  toggleTheme() {
    const body = document.body;
    const themeIcon = document.getElementById("themeIcon");

    if (this.isDarkMode) {
      // Switch to light mode
      body.classList.add("light-mode");
      themeIcon.innerHTML =
        '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"></path>';
      this.isDarkMode = false;
    } else {
      // Switch to dark mode
      body.classList.remove("light-mode");
      themeIcon.innerHTML =
        '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"></path>';
      this.isDarkMode = true;
    }

    // Save theme preference
    localStorage.setItem("migrato-theme", this.isDarkMode ? "dark" : "light");
  }

  loadTheme() {
    const savedTheme = localStorage.getItem("migrato-theme");
    const themeIcon = document.getElementById("themeIcon");

    if (savedTheme === "light") {
      document.body.classList.add("light-mode");
      themeIcon.innerHTML =
        '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"></path>';
      this.isDarkMode = false;
    }
  }

  toggleEditMode() {
    this.editMode = !this.editMode;
    const editToggle = document.getElementById("edit-mode-toggle");

    if (this.editMode) {
      editToggle.innerHTML =
        '<svg class="w-4 h-4 inline mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path></svg>Disable Edit Mode';
      editToggle.className =
        "px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 focus:ring-2 focus:ring-red-500 transition-colors";
      this.enableInlineEditing();
    } else {
      editToggle.innerHTML =
        '<svg class="w-4 h-4 inline mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"></path></svg>Enable Edit Mode';
      editToggle.className =
        "px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 focus:ring-2 focus:ring-blue-500 transition-colors";
      this.disableInlineEditing();
    }
  }

  enableInlineEditing() {
    const cells = document.querySelectorAll('td[data-editable="true"]');
    cells.forEach((cell) => {
      cell.style.cursor = "pointer";
      cell.classList.add("hover:bg-slate-600");
      cell.addEventListener("click", this.handleCellClick.bind(this));
    });
  }

  disableInlineEditing() {
    const cells = document.querySelectorAll('td[data-editable="true"]');
    cells.forEach((cell) => {
      cell.style.cursor = "default";
      cell.classList.remove("hover:bg-slate-600");
      cell.removeEventListener("click", this.handleCellClick.bind(this));
    });
  }

  handleCellClick(event) {
    if (!this.editMode) return;

    const cell = event.target;
    const originalValue = cell.textContent;
    const columnName = cell.getAttribute("data-column");
    const rowId = cell.closest("tr").getAttribute("data-row-id");
    const rowIdValue = cell.closest("tr").getAttribute("data-row-id-value");

    // Create input field
    const input = document.createElement("input");
    input.type = "text";
    input.value = originalValue === "NULL" ? "" : originalValue;
    input.className =
      "w-full px-2 py-1 bg-slate-700 border border-slate-500 rounded text-white focus:ring-2 focus:ring-blue-500 focus:border-blue-500";

    // Replace cell content with input
    cell.innerHTML = "";
    cell.appendChild(input);
    input.focus();
    input.select();

    // Handle save on Enter or blur
    const saveEdit = () => {
      const newValue = input.value.trim();
      const finalValue = newValue === "" ? null : newValue;

      if (finalValue !== (originalValue === "NULL" ? null : originalValue)) {
        this.saveCellEdit(rowId, rowIdValue, columnName, finalValue);
      } else {
        this.restoreCellContent(cell, originalValue);
      }
    };

    const cancelEdit = () => {
      this.restoreCellContent(cell, originalValue);
    };

    input.addEventListener("blur", saveEdit);
    input.addEventListener("keydown", (e) => {
      if (e.key === "Enter") {
        saveEdit();
      } else if (e.key === "Escape") {
        cancelEdit();
      }
    });
  }

  async saveCellEdit(rowId, rowIdValue, columnName, value) {
    try {
      const response = await fetch(`/api/update/${this.currentTable}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          row_id: rowId,
          id_value: rowIdValue,
          data: {
            [columnName]: value,
          },
        }),
      });

      if (response.ok) {
        const result = await response.json();
        this.showNotification("Data updated successfully!", "success");
        // Refresh the table data
        await this.loadTableData();
      } else {
        const error = await response.text();
        this.showNotification("Update failed: " + error, "error");
      }
    } catch (error) {
      console.error("Error saving edit:", error);
      this.showNotification("Update failed: " + error.message, "error");
    }
  }

  restoreCellContent(cell, value) {
    if (value === "NULL") {
      cell.innerHTML = '<span class="text-slate-500 italic">NULL</span>';
    } else {
      cell.textContent = value;
    }
  }

  showNotification(message, type = "info") {
    const notification = document.createElement("div");
    notification.className = `fixed top-4 right-4 px-6 py-3 rounded-lg shadow-lg z-50 transition-all duration-300 transform translate-x-full ${
      type === "success"
        ? "bg-green-600"
        : type === "error"
        ? "bg-red-600"
        : "bg-blue-600"
    } text-white`;
    notification.textContent = message;

    document.body.appendChild(notification);

    // Animate in
    setTimeout(() => {
      notification.classList.remove("translate-x-full");
    }, 100);

    // Remove after 3 seconds
    setTimeout(() => {
      notification.classList.add("translate-x-full");
      setTimeout(() => {
        document.body.removeChild(notification);
      }, 300);
    }, 3000);
  }

  showError(message) {
    const tableView = document.getElementById("tableView");
    tableView.innerHTML =
      '<div class="flex-1 flex items-center justify-center"><div class="text-center"><div class="text-4xl mb-4">ERROR</div><div class="text-red-400 font-medium text-lg">' +
      message +
      "</div></div></div>";
  }

  exportData(format) {
    const url = `/api/export/${this.currentTable}?format=${format}`;
    const link = document.createElement("a");
    link.href = url;
    link.download = `${this.currentTable}.${format}`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    this.showNotification(
      `Exporting ${this.currentTable} as ${format.toUpperCase()}...`,
      "success"
    );
  }

  showImportModal() {
    // Remove existing modal if any
    const existingModal = document.getElementById("importModal");
    if (existingModal) {
      existingModal.remove();
    }
    const modal = document.createElement("div");
    modal.id = "importModal";
    modal.className =
      "fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50";
    modal.innerHTML =
      '<div class="bg-slate-800 rounded-lg p-6 w-96 max-w-full mx-4">' +
      '<div class="flex items-center justify-between mb-4">' +
      '<h3 class="text-lg font-semibold text-white">Import Data</h3>' +
      '<button id="closeImportModal" class="text-slate-400 hover:text-white">X</button>' +
      "</div>" +
      '<form id="importForm" class="space-y-4">' +
      "<div>" +
      '<label class="block text-sm font-medium text-slate-300 mb-2">Format</label>' +
      '<select name="format" class="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white focus:ring-2 focus:ring-blue-500 focus:border-blue-500">' +
      '<option value="csv">CSV</option>' +
      '<option value="json">JSON</option>' +
      '<option value="sql">SQL</option>' +
      "</select>" +
      "</div>" +
      "<div>" +
      '<label class="block text-sm font-medium text-slate-300 mb-2">File</label>' +
      '<input type="file" name="file" accept=".csv,.json,.sql" required class="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white focus:ring-2 focus:ring-blue-500 focus:border-blue-500">' +
      "</div>" +
      '<div class="flex space-x-3 pt-4">' +
      '<button type="submit" class="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 focus:ring-2 focus:ring-blue-500 transition-colors">Import</button>' +
      '<button type="button" id="cancelImport" class="flex-1 px-4 py-2 bg-slate-600 text-white rounded-lg hover:bg-slate-700 focus:ring-2 focus:ring-slate-500 transition-colors">Cancel</button>' +
      "</div>" +
      "</form>" +
      "</div>";
    document.body.appendChild(modal);
    // Close modal handlers
    const closeBtn = modal.querySelector("#closeImportModal");
    const cancelBtn = modal.querySelector("#cancelImport");
    const closeModal = () => modal.remove();
    closeBtn.addEventListener("click", closeModal);
    cancelBtn.addEventListener("click", closeModal);
    modal.addEventListener("click", (e) => {
      if (e.target === modal) closeModal();
    });
    // Handle form submission
    const form = modal.querySelector("#importForm");
    form.addEventListener("submit", (e) => {
      e.preventDefault();
      this.handleImport(form);
    });
  }

  async handleImport(form) {
    const formData = new FormData(form);
    formData.append("table", this.currentTable);
    try {
      const response = await fetch("/api/import/" + this.currentTable, {
        method: "POST",
        body: formData,
      });
      if (response.ok) {
        const result = await response.json();
        this.showNotification(
          `Successfully imported ${
            result.result.inserted_rows || result.result.executed_statements
          } items!`,
          "success"
        );
        document.getElementById("importModal").remove();
        await this.loadTableData();
      } else {
        const error = await response.text();
        this.showNotification("Import failed: " + error, "error");
      }
    } catch (error) {
      console.error("Import error:", error);
      this.showNotification("Import failed: " + error.message, "error");
    }
  }

  async deleteSelectedRows() {
    if (!this.currentTable || this.selectedRows.length === 0) return;
    if (
      !confirm(
        `Delete ${this.selectedRows.length} selected row(s)? This cannot be undone!`
      )
    )
      return;
    try {
      const response = await fetch(
        `/api/table/${this.currentTable}/bulk-delete`,
        {
          method: "DELETE",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ ids: this.selectedRows }),
        }
      );
      if (!response.ok) throw new Error("Failed to delete rows");
      this.selectedRows = [];
      this.loadTableData();
      this.showNotification("Rows deleted successfully!", "success");
    } catch (err) {
      this.showError("Delete failed: " + err.message);
    }
  }
}

document.addEventListener("DOMContentLoaded", () => {
  new MigratoStudio();
});
