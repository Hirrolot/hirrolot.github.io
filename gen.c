#include <assert.h>
#include <errno.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <dirent.h>
#include <unistd.h>

#include <sys/stat.h>
#include <sys/types.h>

#define CONTENT_DIR  "content"
#define OUTPUT_DIR   "posts"
#define EXCELLENT_ME "excellent_me.md"
#define POSTS_MAX    512

#define PANDOC_COMMON_ARGS "--standalone -H header.html"

typedef enum {
    Jan = 1,
    Feb,
    Mar,
    Apr,
    May,
    Jun,
    Jul,
    Aug,
    Sept,
    Oct,
    Nov,
    Dec,
} Month;

typedef struct {
    unsigned day;
    Month month;
    unsigned year;
} PostDate;

typedef struct {
    char *title;
    PostDate date;
} PostMetadata;

static void collect_post_names(size_t *posts_count,
                               char *post_names[POSTS_MAX]);

static void gen_index_md(size_t posts_count,
                         const char *post_names[static posts_count]);
static void gen_posts_history(FILE *index, size_t posts_count,
                              const char *post_names[static posts_count]);

static void gen_target(FILE *makefile, const char *post_name);
static void gen_target_index(FILE *makefile);

static void gen_phony_all(FILE *makefile, size_t posts_count,
                          const char *post_names[static posts_count]);
static void gen_phony_clean(FILE *makefile);

static PostMetadata *
PostMetadata_collect_all(size_t posts_count,
                         const char *post_names[static posts_count]);

static unsigned
PostMetadata_min_year(size_t posts_count,
                      PostMetadata metadata[static posts_count]);
static unsigned
PostMetadata_max_year(size_t posts_count,
                      PostMetadata metadata[static posts_count]);

static Month Month_parse(const char *str);
static const char *Month_str(Month self);

static PostDate PostDate_parse(const char *str);

static PostMetadata PostMetadata_parse(const char *str);
static void PostMetadata_free(PostMetadata *self);

static char *find_post_metadata_field(const char *str, const char *field_name);
static char *find_post_metadata_quoted_field(const char *str,
                                             const char *field_name);

static char *file_base(const char *filename);
static char *read_file_content(const char *filename);

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

static void collect_post_names(size_t *posts_count,
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

    char *excellent_me = read_file_content(EXCELLENT_ME);
    fprintf(index, "%s\n", excellent_me);
    free(excellent_me);

    gen_posts_history(index, posts_count, post_names);

    const bool index_closed = fclose(index) == 0;
    assert(index_closed);
}

static void gen_posts_history(FILE *index, size_t posts_count,
                              const char *post_names[static posts_count]) {
    PostMetadata *metadata = PostMetadata_collect_all(posts_count, post_names);

    const unsigned min_year = PostMetadata_min_year(posts_count, metadata),
                   max_year = PostMetadata_max_year(posts_count, metadata);

    for (unsigned year = max_year; year >= min_year; year--) {
        fprintf(index, "<span class=\"posts-year\" id=\"%u\">%u</span>\n", year,
                year);
        fprintf(index, "<div class=\"posts-history\">\n");

        for (Month month = Dec; month >= Jan; month--) {
            for (unsigned day = 31; day >= 1; day--) {
                for (size_t i = 0; i < posts_count; i++) {
                    if (strcmp(metadata[i].title, "index") == 0) {
                        continue;
                    }

                    if (metadata[i].date.year != year ||
                        metadata[i].date.month != month ||
                        metadata[i].date.day != day) {
                        continue;
                    }

                    fprintf(index,
                            " - [%s](" OUTPUT_DIR "/%s.html)<br><span "
                            "class=\"post-date\">%s %u</span><br>\n",
                            metadata[i].title, post_names[i],
                            Month_str(metadata[i].date.month),
                            metadata[i].date.day);
                }
            }
        }

        fprintf(index, "</div>\n"); // class="posts-history"
    }

    for (size_t i = 0; i < posts_count; i++) {
        PostMetadata_free(&metadata[i]);
    }

    free(metadata);
}

static void gen_target(FILE *makefile, const char *post_name) {
    fprintf(makefile,
            "%s: " CONTENT_DIR "/%s.md\n\t"
            "pandoc " CONTENT_DIR "/%s.md --output " OUTPUT_DIR
            "/%s.html " PANDOC_COMMON_ARGS
            " --table-of-contents --citeproc --css ../style.css "
            "--include-after-body utterances.html --include-in-header "
            "post_header_aux.html\n\n",
            post_name, post_name, post_name, post_name);
}

static void gen_target_index(FILE *makefile) {
    fprintf(makefile,
            "index: " CONTENT_DIR "/index.md\n\t"
            "pandoc " CONTENT_DIR "/index.md " PANDOC_COMMON_ARGS
            " --output index.html --css style.css --include-in-header "
            "index_header_aux.html\n\n");
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

static PostMetadata *
PostMetadata_collect_all(size_t posts_count,
                         const char *post_names[static posts_count]) {
    PostMetadata *metadata = malloc(sizeof metadata[0] * posts_count);
    assert(metadata);

    for (size_t i = 0; i < posts_count; i++) {
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

        metadata[i] = PostMetadata_parse(post_beginning);

        const bool post_fp_closed = fclose(post_fp) == 0;
        assert(post_fp_closed);
    }

    return metadata;
}

static unsigned
PostMetadata_min_year(size_t posts_count,
                      PostMetadata metadata[static posts_count]) {
    unsigned min_year = 9999;

    for (size_t i = 0; i < posts_count; i++) {
        if (strcmp(metadata[i].title, "index") == 0) {
            continue;
        }

        if (min_year > metadata[i].date.year) {
            min_year = metadata[i].date.year;
        }
    }

    return min_year;
}

static unsigned
PostMetadata_max_year(size_t posts_count,
                      PostMetadata metadata[static posts_count]) {
    unsigned max_year = 0;

    for (size_t i = 0; i < posts_count; i++) {
        if (strcmp(metadata[i].title, "index") == 0) {
            continue;
        }

        if (max_year < metadata[i].date.year) {
            max_year = metadata[i].date.year;
        }
    }

    return max_year;
}

#define MONTHS                                                                 \
    ASSOC(Jan);                                                                \
    ASSOC(Feb);                                                                \
    ASSOC(Mar);                                                                \
    ASSOC(Apr);                                                                \
    ASSOC(May);                                                                \
    ASSOC(Jun);                                                                \
    ASSOC(Jul);                                                                \
    ASSOC(Aug);                                                                \
    ASSOC(Sept);                                                               \
    ASSOC(Oct);                                                                \
    ASSOC(Nov);                                                                \
    ASSOC(Dec)

static Month Month_parse(const char *str) {
#define ASSOC(month)                                                           \
    do {                                                                       \
        if (strcmp(str, #month) == 0) {                                        \
            return month;                                                      \
        }                                                                      \
    } while (0)

    MONTHS;

    assert(false);
    return 0;

#undef ASSOC
}

static const char *Month_str(Month self) {
#define ASSOC(month)                                                           \
    case month:                                                                \
        return #month;

    switch (self) {
        MONTHS;
    default:
        assert(false);
        return NULL;
    }

#undef ASSOC
}

#undef MONTHS

static PostDate PostDate_parse(const char *str) {
    PostDate self;

    char *date = find_post_metadata_field(str, "date");

    char month[16];
    const int items_read =
        sscanf(date, "%s %u, %u", month, &self.day, &self.year);
    assert(3 == items_read);

    self.month = Month_parse(month);

    free(date);

    return self;
}

static PostMetadata PostMetadata_parse(const char *str) {
    return (PostMetadata){
        .title = find_post_metadata_quoted_field(str, "title"),
        .date = PostDate_parse(str),
    };
}

static void PostMetadata_free(PostMetadata *self) {
    char *title_start = self->title - strlen("\"");
    free(title_start);
}

static char *find_post_metadata_field(const char *str, const char *field_name) {
    char *start = strstr(str, field_name);
    assert(start);

    start += strlen(field_name);
    assert(':' == start[0]);

    start += strlen(":");
    assert(' ' == start[0]);

    start += strlen(" ");

    char *end = strchr(start, '\n');
    assert(end);

    return strndup(start, end - start);
}

static char *find_post_metadata_quoted_field(const char *str,
                                             const char *field_name) {
    char *start = find_post_metadata_field(str, field_name);
    assert('"' == start[0]);
    start++;

    char *quote_end = strchr(start, '"');
    assert(quote_end);
    quote_end[0] = '\0';

    return start;
}

static char *file_base(const char *filename) {
    return strndup(filename, strchr(filename, '.') - filename);
}

static char *read_file_content(const char *filename) {
    FILE *fp = fopen(filename, "r");
    assert(fp);

    struct stat file_stat;
    const bool stat_succeeded = stat(filename, &file_stat) == 0;
    assert(stat_succeeded);
    assert(file_stat.st_size > 0);

    char *content = malloc(file_stat.st_size);
    assert(content);

    size_t i = 0;
    char c;
    while ((c = fgetc(fp)) != EOF) {
        content[i] = c;
        i++;
    }

    const bool fp_closed = fclose(fp) == 0;
    assert(fp_closed);

    return content;
}
