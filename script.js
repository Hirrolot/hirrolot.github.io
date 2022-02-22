window.addEventListener("load", function(event) {
    moveTocDown();
    createClickableHeaders();
    createPopupNotes();
    genAuthorEmoji();
    decorateToc();
    createCodeAnnotationContainers();
});

function moveTocDown() {
    var introduction = document.querySelector(".introduction");
    var toc = document.querySelector("#TOC");

    toc.parentNode.insertBefore(introduction, toc);
}

function createClickableHeaders() {
    var headers = document.querySelectorAll("h1, h2, h3");

    for (var i = headers.length - 1; i >= 0; i--) {
        var headerLink = document.createElement("a");
        headerLink.className = "header-link";
        headerLink.href = "#" + headers[i].id;
        headerLink.appendChild(headers[i].cloneNode(true));

        headers[i].replaceWith(headerLink);
    }
}

function createPopupNotes() {
    var notes = document.querySelectorAll(".footnote-ref");

    for (var i = notes.length - 1; i >= 0; i--) {
        var sup = document.createElement("sup");
        sup.className = "note-sup";
        sup.textContent = "[^]";

        var fnId = notes[i].href.slice(notes[i].href.lastIndexOf("#") + 1);
        var fnContent = document.getElementById(fnId).firstChild;
        fnContent.querySelector(".footnote-back").remove();

        var noteText = document.createElement("span");
        noteText.className = "note-text";
        noteText.innerHTML = fnContent.innerHTML;

        var noteContainer = document.createElement("span");
        noteContainer.className = "note-container";
        noteContainer.appendChild(sup);
        noteContainer.appendChild(noteText);

        noteContainer.addEventListener("mouseover", function() {
            this.querySelector(".note-text").classList.toggle("show-note");
        });

        noteContainer.addEventListener("mouseout", function() {
            this.querySelector(".note-text").classList.remove("show-note");
        });

        notes[i].replaceWith(noteContainer);
    }

    var footnotes = document.querySelector(".footnotes");
    if (footnotes) {
        footnotes.remove();
    }
}

function genAuthorEmoji() {
    var emojis = [
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

    var randomEmoji = document.createElement("span");
    randomEmoji.textContent = emojis[Math.floor(Math.random() * emojis.length)];
    randomEmoji.innerHTML += "&nbsp;";

    var blogLink = document.createElement("a");
    blogLink.href = "..";
    blogLink.textContent = "Hirrolot's Blog";

    var fancyAuthor = document.createElement("p");
    fancyAuthor.className = "author";
    fancyAuthor.appendChild(randomEmoji);
    fancyAuthor.appendChild(blogLink);

    var author = document.querySelector(".author");

    if (author) {
        author.replaceWith(fancyAuthor);
    }
}

function decorateToc() {
    var toc = document.querySelector("#TOC");

    var tocTitle = document.createElement("h4");
    tocTitle.className = "toc-title";
    tocTitle.textContent = "Contents";

    if (toc) {
        toc.insertBefore(tocTitle, toc.firstChild);
    }
}

function createCodeAnnotationContainers() {
    var codeAnnotations = document.querySelectorAll(".code-annotation");

    for (var i = codeAnnotations.length - 1; i >= 0; i--) {
        var container = document.createElement("div");
        container.className = "code-annotation-container";

        codeAnnotations[i].parentNode.insertBefore(container, codeAnnotations[i]);
        container.appendChild(codeAnnotations[i]);
    }
}