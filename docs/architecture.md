# Architecture


## Principles

1. Separation of concerns.
2. AI providers are replaceable.
3. Git operations are isolated.
4. No destructive actions without confirmation.
5. Every feature must be testable.


---

# Architecture


                 CLI

                  |

              Commands

                  |

        --------------------

        |                  |

       Git                AI

        |                  |

 Repository        AI Providers



---

# Components


## CLI Layer

Responsible for:

- parsing commands
- user interaction
- output formatting


---

## Git Layer


Responsible for:

- git status
- git diff
- git log
- git commit
- git staging


The Git layer must not know about AI.


---

## AI Layer


Responsible for:

- prompt creation
- provider communication
- response parsing


---

## Config Layer


Responsible for:

- API keys
- provider selection
- preferences


---

# Project Structure


cmd/

    gitx.go


internal/

    commands/

    git/

    ai/

    prompts/

    config/

    ui/

    cache/


pkg/

    types/