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

#define EXCELLENT_ME                                                           \
    "<div class=\"me\">"                                                       \
    "<img src=\"Austin-Powers.jpeg\" width=300px />"                           \
    "<div><p class=\"about-me\">I'm a 16 y/o software engineer, most known "   \
    "for my work on [teloxide] and preprocessor metaprogramming: "             \
    "[Metalang99], [Datatype99], and [Interface99]. This is my blog about "    \
    "programming and all the stuff.<br><br></p>" BADGES "</div>"               \
    "</div>\n"                                                                 \
    "[teloxide]: https://github.com/teloxide/teloxide\n"                       \
    "[Metalang99]: https://github.com/Hirrolot/metalang99\n"                   \
    "[Datatype99]: https://github.com/Hirrolot/datatype99\n"                   \
    "[Interface99]: https://github.com/Hirrolot/interface99\n"                 \
    "<hr>\n"

#define BADGES                                                                 \
    "<div class=\"badges\">" GITHUB_BADGE TWITTER_BADGE REDDIT_BADGE           \
    "<br>" TELEGRAM_BADGE GMAIL_BADGE "</div>"

#define GITHUB_BADGE                                                           \
    "<a href=\"https://github.com/Hirrolot\"><img "                            \
    "src=\"https://img.shields.io/badge/"                                      \
    "GitHub-100000?style=for-the-badge&logo=github&logoColor=white\" "         \
    "/></a>"

#define TWITTER_BADGE                                                          \
    "<a href=\"https://twitter.com/hirrolot\"><img "                           \
    "src=\"https://img.shields.io/badge/"                                      \
    "Twitter-1DA1F2?style=for-the-badge&logo=twitter&logoColor=white\" "       \
    "/></a>"

#define REDDIT_BADGE                                                           \
    "<a href=\"https://www.reddit.com/user/Hirrolot/\"><img "                  \
    "src=\"https://img.shields.io/badge/"                                      \
    "Reddit-FF4500?style=for-the-badge&logo=reddit&logoColor=white\" "         \
    "/></a>"

#define TELEGRAM_BADGE                                                         \
    "<a href=\"https://t.me/hirrolot\"><img "                                  \
    "src=\"https://img.shields.io/badge/"                                      \
    "Telegram-2CA5E0?style=for-the-badge&logo=telegram&logoColor=white\" "     \
    "/></a>"

#define GMAIL_BADGE                                                            \
    "<a href=\"https://mailto:hirrolot@gmail.com\"><img "                      \
    "src=\"https://img.shields.io/badge/"                                      \
    "Gmail-D14836?style=for-the-badge&logo=gmail&logoColor=white\" "           \
    "/></a>"

static void collect_post_names(size_t *restrict posts_count,
                               char *post_names[POSTS_MAX]);

static void gen_index_md(size_t posts_count,
                         const char *post_names[static posts_count]);

static void gen_target(FILE *makefile, const char *post_name);
static void gen_target_index(FILE *makefile);

static void gen_phony_all(FILE *makefile, size_t posts_count,
                          const char *post_names[static posts_count]);
static void gen_phony_clean(FILE *makefile);

static char *find_post_title(const char *str);
static char *find_post_date(const char *str);

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
    fprintf(index, "%s", EXCELLENT_ME);

    for (size_t i = 0; i < posts_count; i++) {
        if (strcmp(post_names[i], "index") == 0) {
            continue;
        }

        char post_path[128] = {0};
        snprintf(post_path, sizeof post_path, CONTENT_DIR "/%s.md",
                 post_names[i]);

        FILE *post_fp = fopen(post_path, "r");
        assert(post_fp);

        char post_beginning[512] = {0};
        const size_t chars_read =
            fread(post_beginning, sizeof post_beginning[0],
                  sizeof post_beginning - 1, post_fp);
        assert(sizeof post_beginning - 1 == chars_read);

        char *title = find_post_title(post_beginning),
             *date = find_post_date(post_beginning);

        fprintf(index, " - [%s](" OUTPUT_DIR "/%s.html)<br>%s<br>\n", title,
                post_names[i], date);

        free(title);
        free(date);

        const bool post_fp_closed = fclose(post_fp) == 0;
        assert(post_fp_closed);
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

static char *find_post_title(const char *str) {
    char *title_start = strstr(str, "title: \"");
    assert(title_start);
    title_start += strlen("title: \"");

    char *title_end = strchr(title_start, '\"');
    assert(title_end);

    return strndup(title_start, title_end - title_start);
}

static char *find_post_date(const char *str) {
    char *date_start = strstr(str, "date: ");
    assert(date_start);
    date_start += strlen("date: ");

    char *date_end = strchr(date_start, '\n');
    assert(date_end);

    return strndup(date_start, date_end - date_start);
}

static char *file_base(const char *filename) {
    return strndup(filename, strchr(filename, '.') - filename);
}
