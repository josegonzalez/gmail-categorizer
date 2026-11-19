// Progressive enhancement for gmail-categorizer's web UI. Every page works
// without this file: it adds selection state, archive confirmation and
// keyboard shortcuts.
(function () {
  "use strict";

  document.documentElement.classList.add("js");

  var rows = Array.prototype.slice.call(document.querySelectorAll("[data-row]"));
  var focused = -1;

  function rowChecks() {
    return Array.prototype.slice.call(document.querySelectorAll("[data-row-check]"));
  }

  function checkedRows() {
    return rowChecks().filter(function (box) { return box.checked; });
  }

  function updateSelection() {
    var boxes = rowChecks();
    var checked = checkedRows();
    boxes.forEach(function (box) {
      var row = box.closest("[data-row]");
      if (row) {
        row.classList.toggle("checked", box.checked);
      }
    });

    var all = document.querySelector("[data-select-all]");
    if (all) {
      all.checked = boxes.length > 0 && checked.length === boxes.length;
      all.indeterminate = checked.length > 0 && checked.length < boxes.length;
    }

    var label = document.querySelector("[data-archive-label]");
    if (label) {
      label.textContent = checked.length > 0 ? "Archive " + checked.length : "Archive";
    }
    var button = document.querySelector("#archive-form [data-archive-button]");
    if (button) {
      button.setAttribute("aria-disabled", checked.length === 0 ? "true" : "false");
    }
    document.body.classList.toggle("has-selection", checked.length > 0);
  }

  function setFocus(index) {
    if (rows.length === 0) {
      return;
    }
    index = Math.max(0, Math.min(rows.length - 1, index));
    if (focused >= 0 && rows[focused]) {
      rows[focused].classList.remove("focused");
    }
    focused = index;
    rows[focused].classList.add("focused");
    rows[focused].scrollIntoView({ block: "nearest" });
  }

  function focusedRow() {
    return focused >= 0 ? rows[focused] : null;
  }

  function go(selector, attribute) {
    var el = document.querySelector(selector);
    if (!el) {
      return false;
    }
    var href = attribute ? el.getAttribute(attribute) : el.getAttribute("href");
    if (href) {
      window.location.href = href;
      return true;
    }
    return false;
  }

  // Confirm before any archive, as the TUI does.
  function confirmArchive(form) {
    var boxes = form.querySelectorAll("[data-row-check]");
    if (boxes.length > 0) {
      var checked = Array.prototype.filter.call(boxes, function (box) { return box.checked; });
      if (checked.length === 0) {
        return false;
      }
      var noun = checked.length === 1 ? "grouping" : "groupings";
      return window.confirm("Archive " + checked.length + " " + noun + "?");
    }
    var address = form.querySelector("input[name=address]");
    return window.confirm("Archive every message in " + (address ? address.value : "this grouping") + "?");
  }

  Array.prototype.forEach.call(document.querySelectorAll("[data-archive-form]"), function (form) {
    form.addEventListener("submit", function (event) {
      if (!confirmArchive(form)) {
        event.preventDefault();
      }
    });
  });

  rowChecks().forEach(function (box) {
    box.addEventListener("change", updateSelection);
  });

  var selectAll = document.querySelector("[data-select-all]");
  if (selectAll) {
    selectAll.addEventListener("change", function () {
      rowChecks().forEach(function (box) { box.checked = selectAll.checked; });
      updateSelection();
    });
  }

  var dismiss = document.querySelector("[data-dismiss]");
  if (dismiss) {
    dismiss.addEventListener("click", function () {
      var flash = dismiss.closest("[data-flash]");
      if (flash) {
        flash.remove();
      }
    });
  }

  var sortSelect = document.querySelector("[data-sort-select]");
  if (sortSelect) {
    sortSelect.addEventListener("change", function () {
      sortSelect.form.submit();
    });
  }

  document.addEventListener("keydown", function (event) {
    if (event.defaultPrevented || event.ctrlKey || event.metaKey || event.altKey) {
      return;
    }
    var target = event.target;
    var tag = target && target.tagName;
    if (tag === "INPUT" || tag === "SELECT" || tag === "TEXTAREA" || (target && target.isContentEditable)) {
      return;
    }

    var row = focusedRow();
    switch (event.key) {
      case "j":
        if (!go("[data-next]")) {
          setFocus(focused + 1);
        }
        break;
      case "k":
        if (!go("[data-prev]")) {
          setFocus(focused < 0 ? 0 : focused - 1);
        }
        break;
      case "x":
        if (row) {
          var box = row.querySelector("[data-row-check]");
          if (box) {
            box.checked = !box.checked;
            updateSelection();
          }
        }
        break;
      case "o":
      case "Enter":
        if (event.key === "Enter" && target !== document.body) {
          return;
        }
        if (row) {
          var link = row.querySelector("[data-row-link]");
          if (link) {
            window.location.href = link.getAttribute("href");
          }
        }
        break;
      case "e":
        var form = document.querySelector("[data-archive-form]");
        if (!form) {
          return;
        }
        if (row && checkedRows().length === 0) {
          var rowBox = row.querySelector("[data-row-check]");
          if (rowBox) {
            rowBox.checked = true;
            updateSelection();
          }
        }
        if (confirmArchive(form)) {
          form.submit();
        }
        break;
      case "f":
        go("[data-filter-toggle]", "data-filter-toggle");
        break;
      case "s":
        go("[data-sort-next]");
        break;
      case "u":
        go("[data-back]");
        break;
      default:
        return;
    }
    event.preventDefault();
  });

  updateSelection();
})();
