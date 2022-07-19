window.addEventListener("load", function (event) {
    moveTocDown();
    createClickableHeaders();
    createPopupNotes();
    genAuthorEmoji();
    decorateToc();
    createCodeAnnotationContainers();
});

function moveTocDown() {
    const introduction = document.querySelector(".introduction");
    const toc = document.querySelector("#TOC");

    if (toc && introduction) {
        toc.parentNode.insertBefore(introduction, toc);

        // Previously it was "none". After we are done moving it, we make it visible.
        toc.style.display = "block";
    }
}

function createClickableHeaders() {
    const headers = document.querySelectorAll("h1:not(.blog-title), h2, h3");

    for (var i = headers.length - 1; i >= 0; i--) {
        const headerLink = document.createElement("a");
        headerLink.className = "header-link";
        headerLink.href = "#" + headers[i].id;
        headerLink.appendChild(headers[i].cloneNode(true));

        headers[i].replaceWith(headerLink);
    }
}

function createPopupNotes() {
    // Pop-up notes overflow screen size sometimes. This is a hack, I am noob at CSS/JS.
    // TODO: Adjust notes dynamically.
    if (window.innerWidth <= 600) {
        return;
    }

    const notes = document.querySelectorAll(".footnote-ref");

    for (var i = notes.length - 1; i >= 0; i--) {
        const sup = document.createElement("sup");
        sup.className = "note-sup";
        sup.textContent = "[^]";

        const fnId = notes[i].href.slice(notes[i].href.lastIndexOf("#") + 1);
        const fnContent = document.getElementById(fnId).firstChild;
        fnContent.querySelector(".footnote-back").remove();

        const noteText = document.createElement("span");
        noteText.className = "note-text";
        noteText.innerHTML = fnContent.innerHTML;

        const noteContainer = document.createElement("span");
        noteContainer.className = "note-container";
        noteContainer.appendChild(sup);
        noteContainer.appendChild(noteText);

        noteContainer.addEventListener("mouseover", function () {
            this.querySelector(".note-text").classList.toggle("show-note");
        });

        noteContainer.addEventListener("mouseout", function () {
            this.querySelector(".note-text").classList.remove("show-note");
        });

        notes[i].replaceWith(noteContainer);
    }

    const footnotes = document.querySelector(".footnotes");
    if (footnotes) {
        footnotes.remove();
    }
}

function genAuthorEmoji() {
    const emojis = [
        "💋",
        "💄",
        "💅",
        "❤️",
        "👄",
        "👅",
        "👠",
        "🌹",
        "🌺",
        "🌷",
        "💐",
        "🥀",
        "🍄",
        "🌹",
        "🍎",
        "🍒",
        "🍓",
        "🍇",
        "🍉",
        "🍌",
        "🍫",
        "🍭",
        "🍬",
        "🍧",
        "🍝",
        "🎈",
        "💕",
        "💖",
        "💘",
        "💞",
        "💓",
        "💝",
        "💗",
        "🎀",
        "🎁",
        "🌈",
        "🦄",
    ];

    const randomEmoji = document.createElement("span");
    randomEmoji.textContent = emojis[Math.floor(Math.random() * emojis.length)];
    randomEmoji.innerHTML += "&nbsp;";

    const blogLink = document.createElement("a");
    blogLink.href = "..";
    blogLink.textContent = "Hirrolot's Blog";

    const fancyAuthor = document.createElement("p");
    fancyAuthor.className = "author";
    fancyAuthor.appendChild(randomEmoji);
    fancyAuthor.appendChild(blogLink);

    const author = document.querySelector(".author");

    if (author) {
        author.replaceWith(fancyAuthor);
    }
}

function decorateToc() {
    const toc = document.querySelector("#TOC");

    const tocTitle = document.createElement("h4");
    tocTitle.className = "toc-title";
    tocTitle.textContent = "Table of Contents";

    if (toc) {
        toc.insertBefore(tocTitle, toc.firstChild);
    }
}

function createCodeAnnotationContainers() {
    const codeAnnotations = document.querySelectorAll(".code-annotation");

    for (var i = codeAnnotations.length - 1; i >= 0; i--) {
        const container = document.createElement("div");
        container.className = "code-annotation-container";

        codeAnnotations[i].parentNode.insertBefore(container, codeAnnotations[i]);
        container.appendChild(codeAnnotations[i]);
    }
}
