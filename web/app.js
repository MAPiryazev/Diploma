const API_BASE = "http://localhost:8080";

document.addEventListener("DOMContentLoaded", () => {
  checkHealth();

  const addForm = document.getElementById("addForm");
  addForm.addEventListener("submit", addTransaction);

  const editForm = document.getElementById("editForm");
  editForm.addEventListener("submit", updateTransaction);

  const typeSelect = document.getElementById("type");
  typeSelect.addEventListener("change", () => applyAccountRules(typeSelect.value));

  const editTypeSelect = document.getElementById("editType");
  editTypeSelect.addEventListener("change", () => applyEditAccountRules(editTypeSelect.value));

  applyAccountRules(typeSelect.value);
  applyEditAccountRules(editTypeSelect.value);
});

function setServerStatus(text, color) {
  const el = document.getElementById("serverStatus");
  el.textContent = text;
  el.style.color = color;
}

function checkHealth() {
  fetch(`${API_BASE}/health`)
    .then((r) => r.json())
    .then(() => setServerStatus("Server is OK", "#27ae60"))
    .catch(() => setServerStatus("Server is unavailable", "#e74c3c"));
}

function applyAccountRules(type) {
  const fromRow = document.getElementById("fromAccountRow");
  const toRow = document.getElementById("toAccountRow");

  const fromInput = document.getElementById("fromAccountId");
  const toInput = document.getElementById("toAccountId");

  // reset required
  fromInput.required = false;
  toInput.required = false;

  // show/hide + required
  if (type === "income") {
    fromRow.classList.add("hidden");
    toRow.classList.remove("hidden");
    toInput.required = true;
  } else if (type === "expense") {
    fromRow.classList.remove("hidden");
    toRow.classList.add("hidden");
    fromInput.required = true;
  } else if (type === "transfer") {
    fromRow.classList.remove("hidden");
    toRow.classList.remove("hidden");
    fromInput.required = true;
    toInput.required = true;
  } else {
    fromRow.classList.remove("hidden");
    toRow.classList.remove("hidden");
  }
}

function applyEditAccountRules(type) {
  const fromRow = document.getElementById("editFromAccountRow");
  const toRow = document.getElementById("editToAccountRow");

  const fromInput = document.getElementById("editFromAccountId");
  const toInput = document.getElementById("editToAccountId");

  fromInput.required = false;
  toInput.required = false;

  if (type === "income") {
    fromRow.classList.add("hidden");
    toRow.classList.remove("hidden");
    toInput.required = true;
  } else if (type === "expense") {
    fromRow.classList.remove("hidden");
    toRow.classList.add("hidden");
    fromInput.required = true;
  } else if (type === "transfer") {
    fromRow.classList.remove("hidden");
    toRow.classList.remove("hidden");
    fromInput.required = true;
    toInput.required = true;
  } else {
    fromRow.classList.remove("hidden");
    toRow.classList.remove("hidden");
  }
}

function addTransaction(e) {
  e.preventDefault();

  const occurredAt = document.getElementById("occurredAt").value;
  const type = document.getElementById("type").value;

  const payload = {
    user_id: document.getElementById("userId").value,
    amount: document.getElementById("amount").value,
    currency: document.getElementById("currency").value,
    type,
    status: document.getElementById("txStatus").value,
    occurred_at: new Date(occurredAt).toISOString(),
    description: normalizeOptional(document.getElementById("description").value),

    // accounts
    from_account_id: normalizeOptional(document.getElementById("fromAccountId").value),
    to_account_id: normalizeOptional(document.getElementById("toAccountId").value),
  };

  // UX: clear unused fields (не обязательно, но удобнее)
  if (type === "income") {
    payload.from_account_id = null;
  }
  if (type === "expense") {
    payload.to_account_id = null;
  }

  fetch(`${API_BASE}/items`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  })
    .then((r) => r.json())
    .then((data) => {
      if (data.status === "success") {
        alert("Transaction created");
        document.getElementById("addForm").reset();
        applyAccountRules(document.getElementById("type").value);

        if (document.getElementById("listUserId").value) {
          listTransactions();
        }
        return;
      }
      alert(`Request error: ${data.message || "unknown error"}`);
    })
    .catch((err) => alert(`Network error: ${err.message}`));
}

function listTransactions() {
  const userId = document.getElementById("listUserId").value;
  if (!userId.trim()) {
    alert("user_id is required");
    return;
  }

  fetch(`${API_BASE}/items?user_id=${encodeURIComponent(userId)}`)
    .then((r) => r.json())
    .then((data) => {
      if (data.status === "error") {
        alert(`Request error: ${data.message || "unknown error"}`);
        return;
      }

      const tbody = document.getElementById("tableBody");
      tbody.innerHTML = "";

      const rows = Array.isArray(data.data) ? data.data : [];
      if (rows.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" class="text-center">No data</td></tr>';
        document.getElementById("tableContainer").classList.remove("hidden");
        return;
      }

      rows.forEach((tx) => {
        const row = document.createElement("tr");
        const date = new Date(tx.occurred_at).toLocaleString("ru-RU");

        row.innerHTML = `
          <td title="${escapeHtml(tx.id)}">${escapeHtml(tx.id.slice(0, 8))}...</td>
          <td>${escapeHtml(String(tx.amount))}</td>
          <td>${escapeHtml(tx.currency)}</td>
          <td>${escapeHtml(tx.type)}</td>
          <td><span class="status-badge ${escapeHtml(tx.status)}">${escapeHtml(tx.status)}</span></td>
          <td>${escapeHtml(date)}</td>
          <td class="actions">
            <button class="btn btn-sm btn-edit" data-action="edit">Edit</button>
            <button class="btn btn-sm btn-delete" data-action="delete">Delete</button>
          </td>
        `;

        row.querySelector('[data-action="edit"]').addEventListener("click", () => openEditModal(tx));
        row
          .querySelector('[data-action="delete"]')
          .addEventListener("click", () => deleteTransaction(tx.id, tx.user_id));

        tbody.appendChild(row);
      });

      document.getElementById("tableContainer").classList.remove("hidden");
    })
    .catch((err) => alert(`Network error: ${err.message}`));
}

function openEditModal(tx) {
  const dateStr = new Date(tx.occurred_at).toISOString().slice(0, 16);

  document.getElementById("editId").value = tx.id;
  document.getElementById("editUserId").value = tx.user_id;

  document.getElementById("editAmount").value = tx.amount;
  document.getElementById("editCurrency").value = tx.currency;
  document.getElementById("editType").value = tx.type;
  document.getElementById("editStatus").value = tx.status;

  document.getElementById("editFromAccountId").value = tx.from_account_id || "";
  document.getElementById("editToAccountId").value = tx.to_account_id || "";

  document.getElementById("editOccurredAt").value = dateStr;

  applyEditAccountRules(tx.type);
  document.getElementById("editModal").classList.remove("hidden");
}

function closeEditModal() {
  document.getElementById("editModal").classList.add("hidden");
}

window.closeEditModal = closeEditModal;
window.listTransactions = listTransactions;
window.getAnalytics = getAnalytics;

function updateTransaction(e) {
  e.preventDefault();

  const id = document.getElementById("editId").value;
  const userId = document.getElementById("editUserId").value;
  const occurredAt = document.getElementById("editOccurredAt").value;
  const type = document.getElementById("editType").value;

  const payload = {
    id,
    user_id: userId,
    amount: document.getElementById("editAmount").value,
    currency: document.getElementById("editCurrency").value,
    type,
    status: document.getElementById("editStatus").value,
    occurred_at: new Date(occurredAt).toISOString(),

    from_account_id: normalizeOptional(document.getElementById("editFromAccountId").value),
    to_account_id: normalizeOptional(document.getElementById("editToAccountId").value),
  };

  if (type === "income") {
    payload.from_account_id = null;
  }
  if (type === "expense") {
    payload.to_account_id = null;
  }

  fetch(`${API_BASE}/items/${encodeURIComponent(id)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  })
    .then((r) => r.json())
    .then((data) => {
      if (data.status === "success") {
        alert("Transaction updated");
        closeEditModal();
        listTransactions();
        return;
      }
      alert(`Request error: ${data.message || "unknown error"}`);
    })
    .catch((err) => alert(`Network error: ${err.message}`));
}

function deleteTransaction(id, userId) {
  const ok = confirm("Delete transaction?");
  if (!ok) return;

  fetch(`${API_BASE}/items/${encodeURIComponent(id)}?user_id=${encodeURIComponent(userId)}`, {
    method: "DELETE",
  })
    .then((r) => r.json())
    .then((data) => {
      if (data.status === "success") {
        alert("Transaction deleted");
        listTransactions();
        return;
      }
      alert(`Request error: ${data.message || "unknown error"}`);
    })
    .catch((err) => alert(`Network error: ${err.message}`));
}

function getAnalytics() {
  const userId = document.getElementById("analyticsUserId").value;
  const from = document.getElementById("analyticsFrom").value;
  const to = document.getElementById("analyticsTo").value;

  if (!userId.trim() || !from || !to) {
    alert("user_id, from and to are required");
    return;
  }

  const fromISO = new Date(from).toISOString();
  const toISO = new Date(to).toISOString();

  fetch(
    `${API_BASE}/analytics?user_id=${encodeURIComponent(userId)}&from=${encodeURIComponent(fromISO)}&to=${encodeURIComponent(
      toISO
    )}`
  )
    .then((r) => r.json())
    .then((data) => {
      if (data.status === "error") {
        alert(`Request error: ${data.message || "unknown error"}`);
        return;
      }

      document.getElementById("statSum").textContent = data?.data?.sum ?? "-";
      document.getElementById("statAvg").textContent = data?.data?.avg ?? "-";
      document.getElementById("statCount").textContent = data?.data?.count ?? "-";
      document.getElementById("statMedian").textContent = data?.data?.median ?? "-";
      document.getElementById("statP90").textContent = data?.data?.percentile_90 ?? "-";

      document.getElementById("analyticsResult").classList.remove("hidden");
    })
    .catch((err) => alert(`Network error: ${err.message}`));
}

function normalizeOptional(s) {
  const v = String(s || "").trim();
  return v === "" ? null : v;
}

function escapeHtml(s) {
  return String(s)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}
