# FAQ

??? question "Will it delete my existing images or volumes?"
    No. Everything present before the session starts is never removed, and
    nothing in use by other containers is deleted.

??? question "Where does my data go?"
    Session volumes are deleted with the session. Use a bind mount
    (`-v ./data:/data`) or an existing volume to keep data.

??? question "The plugin crashed. Is something left behind?"
    It is still tracked: `docker ephemeral ls` shows it and `gc` or `down`
    removes it.

??? question "Does it work with Docker Desktop and remote contexts?"
    It works with anything the `docker` CLI talks to. Day-to-day testing
    happens on Linux.

??? question "Is it stable?"
    It is pre-1.0: covered by unit and integration tests, but flags and labels
    may still change between minor versions.
