#include <assert.h>
#include <errno.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <dirent.h>

#define CONTENT_DIR "content"
#define OUTPUT_DIR  "posts"
#define PANDOC_ARGS "--standalone -H header.html --toc"

#define POSTS_MAX 512

static void collect_post_names(size_t *restrict posts_count,
                               char *post_names[POSTS_MAX]);

static void gen_index_md(size_t posts_count,
                         const char *post_names[static posts_count]);

static void gen_target(FILE *makefile, const char *post_name);
static void gen_target_index(FILE *makefile);

static void gen_phony_all(FILE *makefile, size_t posts_count,
                          const char *post_names[static posts_count]);
static void gen_phony_clean(FILE *makefile);

static char *file_base(const char *filename);

int main(void) {
    FILE *makefile = fopen("Makefile", "w");
    assert(makefile);

    fprintf(makefile, ".PHONY: all clean\n\n");

    char *post_names[POSTS_MAX] = {0};
    size_t posts_count = 0;
    collect_post_names(&posts_count, post_names);

    for (size_t i = 0; i < posts_count; i++) {
        gen_target(makefile, post_names[i]);
    }

    gen_target_index(makefile);
    gen_phony_all(makefile, posts_count, (const char **)post_names);
    gen_phony_clean(makefile);

    gen_index_md(posts_count, (const char **)post_names);

    for (size_t i = 0; i < posts_count; i++) {
        free(post_names[i]);
    }

    const bool makefile_closed = fclose(makefile) == 0;
    assert(makefile_closed);
}

static void collect_post_names(size_t *restrict posts_count,
                               char *post_names[POSTS_MAX]) {
    DIR *dir = opendir(CONTENT_DIR);
    assert(dir);

    while (true) {
        errno = 0;
        struct dirent *entry = readdir(dir);
        if (NULL == entry) {
            assert(0 == errno);
            break;
        }

        if (strcmp(entry->d_name, ".") == 0 ||
            strcmp(entry->d_name, "..") == 0 ||
            strcmp(entry->d_name, "index.md") == 0) {
            continue;
        }

        char *post_name = file_base(entry->d_name);
        post_names[*posts_count] = post_name;
        (*posts_count)++;
    }

    const bool dir_closed = closedir(dir) == 0;
    assert(dir_closed);
}

static void gen_index_md(size_t posts_count,
                         const char *post_names[static posts_count]) {
    FILE *index = fopen(CONTENT_DIR "/index.md", "w");
    assert(index);

    fprintf(index, "---\n"
                   "title: Hirrolot's Blog\n"
                   "---\n\n");

    for (size_t i = 0; i < posts_count; i++) {
        if (strcmp(post_names[i], "index") != 0) {
            fprintf(index, " - [%s](" OUTPUT_DIR "/%s.html)\n", post_names[i],
                    post_names[i]);
        }
    }

    const bool index_closed = fclose(index) == 0;
    assert(index_closed);
}

static void gen_target(FILE *makefile, const char *post_name) {
    fprintf(makefile,
            "%s: " CONTENT_DIR "/%s.md\n\t"
            "pandoc " CONTENT_DIR "/%s.md -o " OUTPUT_DIR
            "/%s.html " PANDOC_ARGS " --css ../style.css\n\n",
            post_name, post_name, post_name, post_name);
}

static void gen_target_index(FILE *makefile) {
    fprintf(makefile,
            "index: " CONTENT_DIR "/index.md\n\t"
            "pandoc " CONTENT_DIR "/index.md -o index.html " PANDOC_ARGS
            " --css style.css \n\n");
}

static void gen_phony_all(FILE *makefile, size_t posts_count,
                          const char *post_names[static posts_count]) {
    fprintf(makefile, "all: ");

    for (size_t i = 0; i < posts_count; i++) {
        fprintf(makefile, "%s ", post_names[i]);
    }

    fprintf(makefile, "index\n\n");
}

static void gen_phony_clean(FILE *makefile) {
    fprintf(makefile, "clean:\n\trm " OUTPUT_DIR "/*.html\n\n");
}

static char *file_base(const char *filename) {
    return strndup(filename, strchr(filename, '.') - filename);
}
