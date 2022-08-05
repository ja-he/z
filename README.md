# `z`

A simple, git-centric note management system for my personal use.

Entirely ephemeral, totally theoretical, and as of now non-existent.

## What is this?

Hopefully a reliable and well-written recreation of what so far has been a very
hacky and reliably error-prone shell script.

It's supposed to be the system that helps me approach my notes with minimal
mental overhead:

- Simple CLI for common operations (new text note, search by filename, sync, ...)
- Opinionated wherever an opinion makes things easier at note-taking-time
- Integration with my editor
- Built around my preferred tools/formats

## Who is this for?

Just me.
I can't stop you from using it, but I don't think you'll like it too much
because I don't intend in putting any effort into making it likeable.
(I have to like it either way due to [sunk cost fallacy])

[sunk cost fallacy]: <https://en.wikipedia.org/wiki/Sunk_cost#Fallacy_effect>

## Who is the rest of this README for?

Also me :^)
As I'm writing this I haven't started working on the actual thing at all, but I
figure I'll just keep the main planning information in this document.

---

## Shape of the thing

I have a rough idea from the script I've been using, but I also know of some
shortcomings already that I'd like to ameliorate and also could see a proper
tool allowing me to follow more complex idioms in my notes.

This section is part wish-list and part check-list.

### Commands

This is a running list of planned and realized commands that have already been
assigned.

- [ ] `search` invokes FZF for file/text/... search
- [ ] `edit` open the (if necessary newly-created) note for editing
- [ ] `sync` updates with Git
- [ ] `init` set up (e.g. on a new machine)

### Types of content

This is a running list of the types of content I explicitly want to support.

- [ ] Markdown 
  - presumably targeting Pandoc, perhaps sometimes commonmark for Zola?
- [ ] Plain text
- [ ] LaTeX
  - focussed on whatever engine I land on, probably LuaLaTeX
  - for any notes that I want to nicely typeset
  - this makes me think that I might want to keep them in directories,
    so having a note 'my-cool-doc' I don't do 'path/my-cool-doc.tex' but
    instead maybe 'path/my-cool-doc/index.tex' and then I could have a
    'Makefile' alongside it.
- [ ] XournalPP
  - I _love_ taking notes with XournalPP
  - I don't love some of the format (e.g. it un-hides layers when you reopen)
  - here as well, the directory idea could fly nicely:
    whenever I save a note it generates something (e.g. a PNG or SVG) from it
    allowing viewing even in the browser.
- [ ] Images (PNG, JPG, ...)
  - no way around these
  - mostly I use these for insightful graphics I find somewhere
  - they need to have some way of having a source and a title (or a reason why
    I put them there) represented
  - ...again, the directory idea could really work for this
- [ ] Other stuff
  - e.g. JSON, YAML, ...
  - this doesn't need to be explicitly supported by Z but it ought to be
    possible.

### Other requirements

- Configurability for context
  - when I'm at work, I don't want to have my genuinely private notes pop up if
    I search for a file
  - It needs to be possible to _at least_ have context-based omission of
    some content.
  - Right now I use Git submodules so I can actually omit the contents of
    certain categories entirely (just not clone the thing).
  - For me to be able to use this for work, it needs to allow for a submodule
    for the work category (because otherwise that'd get pretty dicey with NDA
    stuff)
