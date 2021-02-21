const Controller = {
  data: {
    memo: {},
    fragments: [],
    options: {
      MatchCase: false,
      MatchWholeWord: false,
      UseRegularExpression: false,
    },
    lineLimit: 20
  },

  toggleOption: (opt) => {
    Controller.data.options[opt] = !Controller.data.options[opt];
    setTimeout(Controller.search, 0);
  },

  search: (ev) => {
    if (ev) {
      ev.preventDefault();
    }
    const form = document.getElementById("form");
    const data = Object.fromEntries(new FormData(form));
    const opts = Controller.data.options;
    const response = fetch(`/search?q=${data.query}&opts=${JSON.stringify(opts)}`).then((response) => {
      response.json().then((results) => {
        Controller.data.fragments = results;
        setTimeout(Controller.updateTable, 0);
      });
    });
  },

  debouncedSearch: () => {},

  debounce: (fn, t) => {
    let timeoutID = null;
    const cb = () => {
      if (timeoutID != null) {
        clearTimeout(timeoutID);
      }
      timeoutID = setTimeout(() => {
        fn();
        timeoutID = null;
      }, t);
    };
    return cb;
  },

  updateTable: () => {
    const table = document.getElementById("fragments-container");
    let content = '';
    for (let i = 0; i < Controller.data.fragments.length; i++) {
      const frag = Controller.data.fragments[i];
      const { lines } = Controller.bookFragment(frag);
      content += `
      <div class="row justify-content-center">
        <div class="fragment-wrapper col-xs-12 col-md-8">
          <div class="fragment-title">Fragment #${i+1}</div>
          <div class="col text-center">
            <button
              title="Load previous ${Controller.data.lineLimit} lines"
              class="btn btn-sm btn-outline-secondary btn-arrow"
              onclick="Controller.loadMore('p', ${i})">
              ↑
            </button>
          </div>
          <div class="book-fragment" id="fragment${i}">${lines}</div>
          <div class="col text-center">
            <button
              title="Load next ${Controller.data.lineLimit} lines"
              class="btn btn-sm btn-outline-secondary btn-arrow"
              onclick="Controller.loadMore('n', ${i})">
              ↓
            </button>
          </div>
        </div>
      </div>
      `;
    }
    table.innerHTML = content;
  },

  loadMore: (k, i) => {
    const frag = Controller.data.fragments[i];
    let ix = 0;
    if (k == "p") {
      if (!frag.Previous || !frag.Previous.length) {
        return;
      }
      ix = frag.Previous[0].StartIndex;
    } else {
      if (!frag.Next || !frag.Next.length) {
        return;
      }
      ix = frag.Next[frag.Next.length-1].EndIndex+1;
    }
    const response = fetch(`/load?k=${k}&ix=${ix}&limit=${Controller.data.lineLimit}`).then((response) => {
      response.json().then((result) => {
        result = (result || []);
        if (k == "p") {
          const previous = Controller.data.fragments[i].Previous || [];
          Controller.data.fragments[i].Previous = [...result, ...previous];
        } else {
          const next = Controller.data.fragments[i].Next || [];
          Controller.data.fragments[i].Next = [...next, ...result];
        }
        const fragEl = document.getElementById(`fragment${i}`);
        const { lines } = Controller.bookFragment(Controller.data.fragments[i]);
        fragEl.innerHTML = lines;
      });
    });
  },

  bookFragment: (result) => {
    const previousLines = result.Previous;
    const matchedLine = result.Matched;
    const nextLines = result.Next;
    let out = "";
    for (let i = 0; i < previousLines.length; i++) {
      const l = previousLines[i];
      out += `${Controller.getText(l.Content)}<br>`
    }
    out += Controller.getTextHighlighted(matchedLine);
    for (let i = 0; i < nextLines.length; i++) {
      const l = nextLines[i];
      out += `${Controller.getText(l.Content)}<br>`
    }
    return {
      lines: out,
    };
  },

  getText: (b) => {
    return Controller.bytes2str(Controller.base64ToBytes(b));
  },

  getTextHighlighted: (l) => {
    const b = Controller.base64ToBytes(l.Content);
    const portion1 = b.slice(l.StartIndex-l.StartIndex, l.MatchedStartIndex-l.StartIndex);
    const portion2 = b.slice(l.MatchedStartIndex-l.StartIndex, l.MatchedEndIndex-l.StartIndex);
    const portion3 = b.slice(l.MatchedEndIndex-l.StartIndex, l.EndIndex-l.StartIndex);
    const reunited = new Uint8Array([
      ...portion1,
      ...Controller.str2bytes(`<span class="match-highlighted">`),
      ...portion2,
      ...Controller.str2bytes(`</span>`),
      ...portion3
    ]);
    const text = Controller.bytes2str(reunited);
    return text;
  },
  
  base64ToBytes: (base64) => {
      const binary_string = window.atob(base64);
      const len = binary_string.length;
      const bytes = new Uint8Array(len);
      for (let i = 0; i < len; i++) {
          bytes[i] = binary_string.charCodeAt(i);
      }
      return bytes;
  },

  bytes2str: (bytes) => {
    const decoder = new TextDecoder();
    return decoder.decode(bytes);
  },

  str2bytes: (str) => {
    if (!Controller.data.memo[str]) {
      const encoder = new TextEncoder();
      Controller.data.memo[str] = encoder.encode(str);
    }
    return Controller.data.memo[str];
  },
};

const form = document.getElementById("form");
form.addEventListener("submit", Controller.search);
Controller.debouncedSearch = Controller.debounce(Controller.search, 500);
