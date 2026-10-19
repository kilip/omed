# Changelog

## [0.3.0](https://github.com/kilip/omed/compare/v0.2.0...v0.3.0) (2026-10-19)


### Features

* **api:** ✨ add chart of accounts seed template and endpoint ([fc3de8e](https://github.com/kilip/omed/commit/fc3de8e597807a55a1df62aabe57dac4f7cefe8b))
* **api:** ✨ add endpoint listing assignable roles ([e1c0e28](https://github.com/kilip/omed/commit/e1c0e285ea190b9effdfc1c75c927dee4a600dd3))
* **api:** ✨ add mailer inspection endpoints to testctl ([a4e0151](https://github.com/kilip/omed/commit/a4e0151e8b1f69eb94bbf0e5c1c903157db6dbf1))
* **api:** ✨ integrate mailing platform mailx and invitation emails ([1217988](https://github.com/kilip/omed/commit/12179889e901f1dbbc36ebbe477417dae8b3ecc8))
* **api:** ✨ require a display name on roles ([3b24176](https://github.com/kilip/omed/commit/3b24176febe78c562320ffc07629d5980b483e8b))
* **api:** ✨ restrict members and invitations in personal workspace ([6124583](https://github.com/kilip/omed/commit/61245834517dca35420d26473a425314b0ee5e17))
* **web:** ✨ add chart of accounts template setup page and flow ([292265f](https://github.com/kilip/omed/commit/292265fb42b4661a090d49aa392ff7ae66b5ce25))
* **web:** ✨ add create workspace page and dashboard button ([fe5f0df](https://github.com/kilip/omed/commit/fe5f0df09a9e8a33a2c72b8399a59ca82c74099c))
* **web:** ✨ add email field and resend action to invitations ([ce42f39](https://github.com/kilip/omed/commit/ce42f39084adaf05791001094d86aa894b74cc7a))
* **web:** ✨ add odoo-style layout components ([d7e0d69](https://github.com/kilip/omed/commit/d7e0d69ba04d2f2f06ed764d43733969bca48825))
* **web:** ✨ add warm cocoa compact theme ([279fe3b](https://github.com/kilip/omed/commit/279fe3be0ccf266bf80a01664167e7e468146887))
* **web:** ✨ add workspace switch button and edit page ([205d88e](https://github.com/kilip/omed/commit/205d88e1848af7dcf785f352c31c7f7f3bf6e5c1))
* **web:** ✨ enforce single role selection per module ([64b1faf](https://github.com/kilip/omed/commit/64b1fafd4193f436519b39e193911cbc1172ab98))
* **web:** ✨ group roles by module and use pages for member editing ([9f1db18](https://github.com/kilip/omed/commit/9f1db18fed6a9eaa32522352c8edca113b7007a3))
* **web:** ✨ hide members and invitations in personal workspace ([aef72a2](https://github.com/kilip/omed/commit/aef72a27048b3821c808c6953fed5aca02d35c60))
* **web:** ✨ move module menu from sidebar to top navbar ([18a6e02](https://github.com/kilip/omed/commit/18a6e02b8e3923d1d1fd5744bb2dc1fc0c0026e6))
* **web:** ✨ pick roles from the api with i18n labels ([49a2e37](https://github.com/kilip/omed/commit/49a2e377ce1ca0434ec4d966d0e9f419d574418c))
* **web:** ✨ recolor logo and app icons to cocoa ([7bc54ee](https://github.com/kilip/omed/commit/7bc54ee163559b0d1dd5b9b804069e104d1980de))
* **web:** ✨ restyle theme after odoo 20 ([1545962](https://github.com/kilip/omed/commit/1545962d154f812a33cca88f57c56b6204b50b20))
* **web:** ✨ switch theme to vivid orange ([2cd802c](https://github.com/kilip/omed/commit/2cd802cca942bcfca28448afd21df284674b9d41))
* **web:** ✨ use control panel and form view in modules ([e1f4d92](https://github.com/kilip/omed/commit/e1f4d928576674c8742b0ca9843fcd87b04c2aeb))


### Bug Fixes

* **api:** 🐛 use owner id for testctl workspace audit context ([7fc1fff](https://github.com/kilip/omed/commit/7fc1fff08ea12febe711b2b7c92c2b009dc3ecde))
* **web:** 🐛 match dark mode font sizes with light mode ([81be2ba](https://github.com/kilip/omed/commit/81be2ba94decb7c4bb23cca4b426e818f6828918))

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
