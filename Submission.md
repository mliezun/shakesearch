# Submission

Heroku App: https://powerful-castle-29357.herokuapp.com/

Try this inputs:
- `Word`
- `Word` + `MatchCase ON`
- `Word` + `MatchCase ON` + `MatchWholeWord ON`
- `romeo.*juliet` + `UseRegularExpression ON`


### Changes Made

- Backend
    - Search is now based on Regular Expressions.
    - Added 3 search options (on/off): MatchCase, MatchWholeWorld, UseRegularExpression. (same options that vscode has for text searching)
    - Results are returned with a new data structure (see `MatchedSearch` struct). Each match returns the line that matched with the indexes where the match was found. Also returns 5 previous and next lines to add some context. The lines text are bytes encoded as Base64.
    - New endpoint api /load that let's you fetch more previous or next lines.
- Frontend
    - Each matched line is represented in a fragment where the matched text is highlighted and you can load more next or previous lines.
    - New buttons to enable/disable search options.
    - The search is performed as the user writes, you don't need to press a button or hit enter.
    - Added bootstrap to make the site responsive.
    - Added a new stylesheet to make the app a little bit prettier.


### Future Improvements

- Merge matched fragments that share some lines. For example: two consecutive lines that match the same search query are returned as independent fragment and the previous and next lines get repeated. This change should be made in the backend and also the frontend.

- Store book information in postgres, that way we could use some advanced text indexing/searching capabilities that postgres has. Proposed schema:
    ```sql
    CREATE TABLE books (
        bookId serial primary key,
        title text not null,
        publication date not null,
        -- other book info (eg: authorId in case we want to add other authors)
    );
    CREATE TABLE book_lines (
        lineNo bigint primary key,
        lineContent text not null
    );
    ```
- UI/UX improvements
    - Add autocomplete functionality to search box.
    - Group results by book.
