There is an annoying issue with IntelliJ IDEA and Go related to imports: IDEA automatically adds new imports, but often
separates them with an empty line or adds to another import group (import group here is a bunch of imports separated
with an empty line). Subsequent run of goimports doesn't help, because it doesn't join import groups.

This is a simple tool which is intended to be run before goimports: it simply joins all import groups. The subsequent
run of goimports makes them more pretty by splitting them to standard three groups: standard library, foreign and local
packages.