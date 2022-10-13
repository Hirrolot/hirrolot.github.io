// This file performs an HTML DOM transformation at build-time.
// Usage: `node transform.js <input-file.html>`.

const jsdom = require("jsdom");
const { JSDOM } = jsdom;

fs = require('fs');

const transformers = [rearrangePostStructure, decorateToc, createClickableHeaders, createCodeAnnotationContainers]

for (let i = 2; i < process.argv.length; i++) {
    const inputFile = process.argv[i];

    JSDOM.fromFile(inputFile).then(dom => {
        transformers.forEach(transformer => transformer(dom.window.document));
        fs.writeFileSync(inputFile, dom.serialize());
    });
}

function rearrangePostStructure(document) {
    const introduction = document.querySelector(".introduction");
    const toc = document.querySelector("#TOC");

    if (toc && introduction) {
        document.body.insertBefore(introduction, toc);

        const postBody = document.createElement("div");
        postBody.className = "post-body";

        for (let i = document.body.children.length - 1; i >= 3; i--) {
            postBody.prepend(document.body.children[i]);
        }

        document.body.appendChild(postBody);
    }
}

function decorateToc(document) {
    const toc = document.querySelector("#TOC");

    const tocTitle = document.createElement("h4");
    tocTitle.className = "toc-title";
    tocTitle.textContent = "Table of Contents";

    if (toc) {
        toc.insertBefore(tocTitle, toc.firstChild);

        // Previously it was "none". After we are done moving it, we make it visible.
        toc.style.display = "block";
    }
}

function createClickableHeaders(document) {
    const headers = document.querySelectorAll("h1:not(.blog-title), h2, h3");

    for (var i = headers.length - 1; i >= 0; i--) {
        const headerLink = document.createElement("a");
        headerLink.className = "header-link";
        headerLink.href = "#" + headers[i].id;
        headerLink.appendChild(headers[i].cloneNode(true));

        headers[i].replaceWith(headerLink);
    }
}

function createCodeAnnotationContainers(document) {
    const codeAnnotations = document.querySelectorAll(".code-annotation");

    for (var i = codeAnnotations.length - 1; i >= 0; i--) {
        const container = document.createElement("div");
        container.className = "code-annotation-container";

        codeAnnotations[i].parentNode.insertBefore(container, codeAnnotations[i]);
        container.appendChild(codeAnnotations[i]);
    }
}
