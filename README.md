# C4 Architecture Diagrams

An implementation of the [structurizr DSL](https://github.com/structurizr/dsl) for generating [C4 diagrams](https://c4model.com) as code.

Goto
- [Grammar Changes](#changes-in-grammar)
   - [Identifiers vs Strings](#identifiers-vs-strings)
   - [Multiline Strings](#multiline-strings)
   - [Comments](#comments)
   - [Semicolons](#terminators-semicolons)
   - [Restrictions on Redefinition](#redefinition)
   - [Tags](#tags)
- [Parser and Runtime changes](#pre-processing-directives-and-modifiers)
- [Navigating the code](#the-codebase)

# Work in Progress
- [ ] Functional Checker
- [ ] Loader
- [ ] Support pre-processing directives

# Changes in Grammar

This implementation of the DSL varries in a few ways

## Identifiers vs Strings

Identifiers and Strings are different and are represented differently internally.

Identifiers name objects, and in the DSL are only used in assignment and relationship definitions, and referenced from views. 
They must be in the form `[a-z][a-zA-Z0-9_-.]*[a-zA-Z0-9]` and cannot be in quotes.

Strings are everything else, and must be in quotes. Accepted quotes are double, single, and backticks. All names, descriptions, tags, and keys/values for properties and perspectives must be quoted.

## Multiline strings

Strings quoted in single backticks gain the additional property of supporting multi-line content.

Multi-line strings are only supported within definition blocks `{ ... }` and so cannot be used in the short form declarations.

Indentation common to the string will be stripped, as well as leading or trailing newlines. Newlines in the string will be otherwise preserved

```javascript
// valid
// though using backticks, it's only a one-line string
softwareSystem 'my name' `must be one-line description'`

// valid
// multiline strings may be used within blocks
softwareSystem 'system 2' {
    description `
        This is a multi-line
        description of a service`
}

/* 
description will be parsed as:

    "This is a multi-line\ndescription of a service"

after leading newlines and common whitespace indentation are removed
*/

// invalid
// cannot break up the declaration line
softwareSystem 'sad system' `this would be
    an invalid declaration` {}
```

## Comments

Using `#` for single line comments is not supported.

Valid comment notations are `//` for excluding the rest of a line, and `/* */` for block comments, which may be multi-line

```javascript
softwareSystem 'foo' // the rest of this line is comment

softwareSystem /*inline and ignored*/ 'bar'

/*
    Block Comment
*/
   a -> b
```

## Terminators (Semicolons)

The formal grammar for the DSL now uses semicolons `;` to separate statements.

However, much like go code, you do not actually need to have them in code, they are inserted by the lexer when appropriate. In general, an identifier or a string followed by a newline has a terminator inserted

With terminators, you may write your code on the same line, so long as you insert semicolons to separate statements

```javascript
// both situations are identitcal to the parser
description "This line omits the terminator"
description "This line includes it";

// identitcal to separating with newlines
name 'c4'; tags 't1' 't2'; technology 'golang';

// invalid
container 'my container' "It's a container"
    {
        technology 'docker probably'
    }
/* 
the lexer will insert a semicolon after the description string
which will terminate the container declaration.
Then the opening block '{' is out of place
*/

// valid
// so opening braces must be on the same line
container 'better container' {
    technology "pixie-dust"
}

// invalid
// same with closing braces
container 'd' {
    description "so many containers" }
/*
closing braces aren't the same as terminators
so this statement is unfinished according to the compiler
*/
```

## Redefinition

In general, defining anything twice is an error.

Most notably, defining a name, comment, tags, or technology in the short-form declaration of an element means you cannot redeclare the element in the body.

```javascript
// invalid
// redefinition of description
container "c1" "A descriptive note" {
    description "Actually I had more to say..."
}
```

## Tags

With the changes in grammar to strings vs identifiers, tags must be defined as strings.

Tags can still either be mutiple string values, separated by whitespace, or comma-separated values inside one string. It is an error to mix styles.

Rules on redefinition also apply, and tags cannot be both on the declaration line, and in an element body.

```javascript
// valid
// equivalent tag declarations
a -> b 'Gets' 'api' 'tag1' 'tag2'
a -> b 'Gets' 'api' 'tag1,tag2'

// invalid
// redefinition of tags
x -> y 'Puts' 'rpc' 'tag1' {
    tags 'tag2'
}

// invalid
// mixed tag notation
x -> y 'Puts' 'rpc' 'tag1' 'tag2,tag3'
```

## UTF-8

The entire system is UTF-8 compatible. Identifiers are still limited to the restricted range of characters, but string values are not.

```javascript
b -> z '🤖' {
    tags '🌴' "🍔"
}
```

# Pre-Processing Directives and Modifiers

This implementation takes the original DSL's concept of `!directives` and modifies them slightly to represent how they're used in the system

Many properties that were directives in the original DSL are simply keywords in this implementation.

As a rule, directives that supplied only information to the final presentation, such as `!adr` and `!docs` have been promoted to keywords. Remaining and new directives are any that require actions being taken during compile-time.

## Pre-processing

Any directives that modify the code before it's parsed, such as including from other files, is marked by `#` pre-processing directives.

Preprocessor directives must be on a single line, and cannot be intermixed with other content.

### Available pre-processing directives

Directive | Description
----------|------------
`#include <path>` | Injects the contents of the given file inline. Path may be relative or absolute to the local filesystem. Failure to read the file causes processing to stop
`#include? <path>` | Same as `#include` but failure to read the file will not stop processing.
`#load https://<url>` | Same as `#include` but fetches from remote hosts.   This only supports `https://` schemes. URI fragments (trailing `#` such as `https://url#foo`) are not included. Files may be cached by the compiler according to their `Cache-Control` headers.
`#load? https://<url>` | Same as `#include?`

```javascript
// includes the contents of ./common/my-container.c4
// as if it were there directly
a = softwareSystem 'foo' {
    #include common/my-container.c4
}
```

# The Codebase

The compiler is divided up into three stages: Lexing, Parsing, and Checking.

## Loading

As the step before any actual compilation starts, the loader is extremely simple.

It loads files, finds and processes pre-processing directives, and then fetches other local or external resources if needed.

The Loader then supplies named byte-streams to the rest of the compiler to act on.

## Lexing

The Lexer is responsible for the first pass over the DSL. It breaks the stream up into lexical elements such as `String`, `Identifier`, or `Relationship->`, and returns them as a stream of tokens.

Tokens do not carry with them the underlying bytes that generated them, since a lot of the time the syntactical purpose of the token is clear with no additional information. What the tokens do carry is location information relative to the source file they came from.

It also differentiates keywords from identifiers. Normally this is a role of the parser, but in this case the lexer handles it.

End of file (EOF) is also a valid token, and is guaranteed to be the last returned from the lexer's token stream. Calling `nextToken` after EOF results in a panic.

### Lexer

The Lexer is in [`compiler/lexer`](./compiler/lexer/)

This lexer is a simple state machine. It starts in the `rootState` and depending what characters it sees, switches to different states to parse them. Every state knows what characters it expects, and what states are allowed to be moved into afterwards. `nil` represents the terminal state. See this in [`lexer/states`](./compiler/lexer/states.go)

The parent object, the `*Lexer` itself provides all the support functionality for consuming runes and producing tokens. States can do one-rune lookaheads, meaning the lexer supports backing the rune stream up one step.

This is mostly used with the support functions like `acceptOne(...)` which requests a token, sees if it's of a supported type, and returns it if it's not.

State fuctions consume the rune stream with properties on the lexer like `lexer.next()` and `lexer.currentRune`.

The lexer keeps track of where it is in the file, and handles differentiating between human-eye oriented line/column position and byte position. State functions generate tokens by calling `lexer.createToken()` and providing what type of token the last `n` runes represent. The lexer handles knowing how many runes have passed since the previous token, and attaches all the needed information to the token.

Errors are also a sort of token, while usually not used for much except reporting errors, in theory a parser can attempt to recorver from an error token gracefully, such as by aborting the current block definition but otherwise continuing.

It is during lexing that terminators ';' are either tokenized if they're written in the DSL, or automatically generated if ellided. Any state may know that a terminator may follow it, and the `spaceWithOptionalTerminatorState` handles lexing up to a point where a terminator should be inserted. A potentially confusing side-effect of this is errors may refer to unexpected terminators and reference lines in the DSL that do not exist. The parser attempts to provide more helpful details to combat this.

## Parsing

The next step is consuming the stream of lexical tokens and extracting meaning from them. This is the role of the Parser.

The parser reads the token stream and recursively constructs the entire workspace model in memory. This model may not be valid, identifiers aren't checked to ensure they refer to real entities yet and such, but it does mean the DSL is syntactically valid.

Parsing also still makes use of the original DSL to extract further meaning. An 'Identifier' token says an identifier starts here, but not what it's called, so the parser looks up the runes at that token's location to assign an identifier value.

### The Parser

The Parser is in [`compiler/parser`](./compiler/parser)

The parser is modelled very similarly to the lexer. The root parser object deals with the incoming token stream, including lookaheads.

The model in the parser is richer than the lexer and has to be able to self-describe in much more detail.

Parsing is a call-stack and recursion based model, rather than a pure state machine. Each entity type has to know what keywords and child types it supports, and call constructors for those entities as needed.

We also provide the `expectationError` object and functions for returning useful messages to human users, since parsing is likely to be one of the steps with the most issues, and describing them well is important.

```
./compile
error parsing workspace:
    > error parsing workspace definition:
    > got Newline or terminator (';') but expected token type '{'
    Line  1  > workspace {
    Line  2  >     model
              ~ ~ ~ ~ ~ ^^ Here
    Line  3  >     {
    Line  4  >      a -> b
    Line  5  >     }
```

### Entities

The DSL is a fairly straightforward language, and many entities share most properties. So the [`entitiy` object](./compiler/parser/entity.go) can do a lot of heavy lifting for us.

`parser.parseShortDeclarationSeq()`, for instance, takes a variable array of pointers and reads across an entity declaration line, filling in any properties declared there. Since most of the properties on the declaration line are optional, it fills only what it can find.

Most of the heavy lifting with reading keywords and filling appropriate structures is handled by `parser.parseEntityBase()`. The caller supplies which keywords are allowed to be acted on by the grammar, and the parse loop does the rest. The function returns with no error when it encounters a valid token that it can't automatically parse, and the caller must handle it. The caller can then choose to resume the base parsing.

## Checking

Checking is the final phase of compilation, and does the most to enhance and verify the model provided by the parser.

It is during this step that identifiers are resolved to ensure they reference valid entities, and implicit relationships are created between entities where applicable.

With this three-pass compiler, the checker can be extremely intelligent. For instance, entities do not need to be declared before they can be referenced. The checker has the complete model available to it, and can look forwards or backwards to resolve a reference.

```javascript
model {
    a = softwareSystem 'sys 1' {
        -> b  // invalid in the original DSL, but handled here
    }
    b = softwareSystem 'sys 2'
}
```

The checker is also where `!directives` are handled. With the most context available to it, directives can be quite powerful in this pass.