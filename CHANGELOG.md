# Changelog

## [0.2.0](https://github.com/kilip/omed/compare/v0.1.0...v0.2.0) (2026-10-14)


### Features

* ✨ add web dashboard, native auth session, and e2e suite ([3889faf](https://github.com/kilip/omed/commit/3889fafe69a7d4ac9cebf577de31f67cf39dc02c))
* **api:** ✨ add test endpoints and session token route ([beb9f2d](https://github.com/kilip/omed/commit/beb9f2d5ea657295d0ec744f73f0f6fedbbee83e))
* **api:** ✨ implement oauth2 and session management ([e37ea35](https://github.com/kilip/omed/commit/e37ea352e5f06c031db54f45f969867a9bb9eea3))
* **api:** ✨ replace jwks with custom session middleware ([f3288c3](https://github.com/kilip/omed/commit/f3288c38b10836ac1f7dcce29d5c006f38171264))
* **api:** add finance module and chart of accounts endpoint :rocket: ([9ff1b61](https://github.com/kilip/omed/commit/9ff1b61313ec3f8920dac1645eb9e5e5c781b631))
* **api:** add soft delete and simplify account search :rocket: ([2c0b472](https://github.com/kilip/omed/commit/2c0b472a1fce98731babbf17c4908633e309015a))
* **api:** add workspace invitations :rocket: ([76bebad](https://github.com/kilip/omed/commit/76bebadb01450a9d241aae0ee153630e733bf985))
* **api:** add workspace mixin to fin account :rocket: ([87bf7c2](https://github.com/kilip/omed/commit/87bf7c2c4f5eb77e6fbd76890ef63033f15ec260))
* **api:** add workspaces endpoint :rocket: ([afcaca3](https://github.com/kilip/omed/commit/afcaca384307bf4bec7c2ea1ceb19f2562f06c77))
* **api:** auto inject workspace id and add query interceptor :rocket: ([9800cf6](https://github.com/kilip/omed/commit/9800cf6be53dc461719d54745ae0da450372231c))
* **api:** automate audit fields via ent mutator hook :rocket: ([3f6e13f](https://github.com/kilip/omed/commit/3f6e13f33a2bdcf334f0abec07c1ae32d84b3e9f))
* **api:** automate delete audit fields via ent mutator hook :rocket: ([e6e09d2](https://github.com/kilip/omed/commit/e6e09d2a53718e6158c00682a71d70cd80f5de44))
* **api:** integrate casbin workspace-scoped rbac :rocket: ([1c50b84](https://github.com/kilip/omed/commit/1c50b84a747ea5f985f0f7daa72a1a035b19ac03))
* **auth:** ✨ add better auth browser client ([4ce29fa](https://github.com/kilip/omed/commit/4ce29fad0e5ca76b9cee5bf152c10f88569cc471))
* **e2e:** ✨ scaffold e2e workspace package ([42aed7c](https://github.com/kilip/omed/commit/42aed7c3493dd054d515e7ed7182ab4c285d48de))
* **web:** ✨ add dashboard shell, login and modules ([00e7e57](https://github.com/kilip/omed/commit/00e7e5773bba1695b65d16c232a9a5371101802d))
* **web:** ✨ fallback auth url to api url ([4b5510f](https://github.com/kilip/omed/commit/4b5510f2b94707151ade1c97aa7c208bbe00c9ca))
* **web:** ✨ integrate tanstack query ([f8ad1bb](https://github.com/kilip/omed/commit/f8ad1bb04dca02abbee897aa3846ef5041d9908a))
* **web:** ✨ scaffold react-router application ([46cba8a](https://github.com/kilip/omed/commit/46cba8a5955250967765065ec85baa541deec10a))


### Bug Fixes

* **api:** 🐛 register pgx driver and inject ent client ([5eec3f0](https://github.com/kilip/omed/commit/5eec3f0aa16ec49531f603bffa4c811663bbd8e2))
* **e2e:** 🐛 load env files and fix dev runner on windows ([ccb1bf7](https://github.com/kilip/omed/commit/ccb1bf7c185a94131763b58c456f0821c18a4b4d))
* **web:** 🐛 hoist definemodule declaration ([f7d01b4](https://github.com/kilip/omed/commit/f7d01b4b0bee6f6d904457cb8c9686e33ce89dd3))


### Dependencies

* **deps:** sync lockfile and go dependencies :wrench: ([625d141](https://github.com/kilip/omed/commit/625d141af4a29f8cfcbade86b546f14f24739a73))

## 0.1.0 (2026-10-13)


### Features

* **api:** add swaggo annotations and generate openapi spec :rocket: ([bc467f4](https://github.com/kilip/omed/commit/bc467f41e2721ea4272b2f4d8dc9d34c64abd5dc))
* **api:** add user base crud endpoint ([d96209b](https://github.com/kilip/omed/commit/d96209be721765f47ab243d1c3b9502bf76a5f47))
* **api:** add user struct validation and tests :rocket: ([ce01581](https://github.com/kilip/omed/commit/ce015810d02d867b687f500a05fb31d8d0058c20))
* **api:** implement user search endpoint :rocket: ([46a709d](https://github.com/kilip/omed/commit/46a709d40716486d9f11162a2d03dfc0a3b32dc4))


### Bug Fixes

* **ci:** change manifest file name ([3d6b044](https://github.com/kilip/omed/commit/3d6b044f9dfe6a8044a4a049679002a15b7b9443))


### Dependencies

* **deps:** specify allowed versions for typescript package :wrench: ([328cf1d](https://github.com/kilip/omed/commit/328cf1d8ff4f90b1c8070320b467022b69985bee))
