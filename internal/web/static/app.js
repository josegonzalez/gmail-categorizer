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

  // Archive progress. With fetch and streams available, the archive form is
  // sent in the background and the server streams one JSON event per line;
  // otherwise the form posts normally and the page waits for the redirect.
  var archiving = false;
  var progress = document.querySelector("[data-progress]");
  var progressBar = document.querySelector("[data-progress-bar]");
  var progressStatus = document.querySelector("[data-progress-status]");
  var canStream = !!(window.fetch && window.ReadableStream && window.TextDecoder && window.URLSearchParams && window.FormData);

  function plural(n, one, many) {
    return n + " " + (n === 1 ? one : many);
  }

  function showProgress(fraction, text) {
    if (progress) {
      progress.hidden = false;
    }
    if (progressBar) {
      var pct = Math.max(2, Math.min(100, Math.round(fraction * 100)));
      progressBar.style.width = pct + "%";
      progressBar.setAttribute("aria-valuenow", String(pct));
    }
    if (progressStatus && text) {
      progressStatus.hidden = false;
      progressStatus.textContent = text;
    }
  }

  function describe(e) {
    if (e.type === "done") {
      return "Archived " + plural(e.messages, "message", "messages") + ". Loading the inbox...";
    }
    var text = "Archiving " + e.doneMessages + " of " + plural(e.messages, "message", "messages");
    if (e.current) {
      text += e.groupings > 1
        ? " (grouping " + Math.min(e.doneGroupings + 1, e.groupings) + " of " + e.groupings + ": " + e.current + ")"
        : " from " + e.current;
    }
    return text;
  }

  function onProgress(e) {
    var fraction = e.messages > 0 ? e.doneMessages / e.messages : (e.groupings > 0 ? e.doneGroupings / e.groupings : 1);
    showProgress(e.type === "done" ? 1 : fraction, describe(e));
  }

  function streamArchive(form) {
    var redirect = "/";
    // Set once the server answers. Before that, nothing has been archived
    // and a failed request falls back to a normal form post.
    var started = false;
    archiving = true;
    document.body.classList.add("archiving");
    showProgress(0, "Starting the archive...");

    fetch(form.getAttribute("action"), {
      method: "POST",
      body: new URLSearchParams(new FormData(form)),
      headers: { Accept: "application/x-ndjson" },
      credentials: "same-origin"
    }).then(function (res) {
      started = true;
      var type = res.headers.get("Content-Type") || "";
      if (!res.ok || type.indexOf("application/x-ndjson") < 0 || !res.body) {
        // The server answered before archiving anything, usually with an
        // error page. Submit normally so the browser shows it.
        archiving = false;
        form.submit();
        return null;
      }

      var reader = res.body.getReader();
      var decoder = new TextDecoder();
      var buffer = "";
      function pump() {
        return reader.read().then(function (chunk) {
          if (chunk.done) {
            return;
          }
          buffer += decoder.decode(chunk.value, { stream: true });
          var newline;
          while ((newline = buffer.indexOf("\n")) >= 0) {
            var line = buffer.slice(0, newline).trim();
            buffer = buffer.slice(newline + 1);
            if (!line) {
              continue;
            }
            var event = JSON.parse(line);
            if (event.redirect) {
              redirect = event.redirect;
            }
            onProgress(event);
          }
          return pump();
        });
      }
      return pump().then(function () { return redirect; });
    }).catch(function () {
      if (!started) {
        archiving = false;
        form.submit();
        return null;
      }
      // The connection dropped, but the archive keeps running on the server;
      // the inbox page loads once it has finished.
      showProgress(1, "Lost the connection. Waiting for the archive to finish...");
      return redirect;
    }).then(function (target) {
      if (target) {
        window.location.href = target;
      }
    });
  }

  Array.prototype.forEach.call(document.querySelectorAll("[data-archive-form]"), function (form) {
    form.addEventListener("submit", function (event) {
      if (archiving || !confirmArchive(form)) {
        event.preventDefault();
        return;
      }
      if (canStream) {
        event.preventDefault();
        streamArchive(form);
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

    if (archiving) {
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
        // requestSubmit runs the submit handler: confirmation and progress.
        if (form.requestSubmit) {
          form.requestSubmit();
        } else if (confirmArchive(form)) {
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
