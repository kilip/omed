# Changelog

## [0.9.0](https://github.com/kilip/omed/compare/v0.8.0...v0.9.0) (2026-11-01)


### Features

* **agent:** ✨ expose models listing endpoints with authz and tests ([d3aa8b2](https://github.com/kilip/omed/commit/d3aa8b25fe0ca254c60ff4d06a264ac086a8fa2f))
* **agent:** ✨ implement ListModels and ListModelsConfig in provider service ([839df08](https://github.com/kilip/omed/commit/839df084c4f84981a86a1185a956a4a384d3680a))
* **api:** ✨ add access token endpoints ([1a68c9c](https://github.com/kilip/omed/commit/1a68c9c6108b79508003fcdcbcd41a1371ecb721))
* **api:** ✨ add account balance insight service ([91d66cb](https://github.com/kilip/omed/commit/91d66cb5af00e8bdcf31da57b61973ff38f3a3e5))
* **api:** ✨ add cashflow search insight ([29a233e](https://github.com/kilip/omed/commit/29a233e7d2397266a969b9087ac15b5ff80b64c4))
* **api:** ✨ add fin period overview insight service ([02391df](https://github.com/kilip/omed/commit/02391dfd6f8bcde9eec60067c46981686840608c))
* **api:** ✨ add github oauthx driver ([d308860](https://github.com/kilip/omed/commit/d3088606cf0b8a0bbb1aee5062d250bf935e35a0))
* **api:** ✨ add google oauthx driver ([5d99614](https://github.com/kilip/omed/commit/5d99614085a60aaddba0e3c2b0780edb21dc23ce))
* **api:** ✨ add ListModels to llmx provider drivers ([f58288f](https://github.com/kilip/omed/commit/f58288fda4bc3e7c3a0868c5273f8db773edfb01))
* **api:** ✨ add mcp tool registrar on the go sdk ([bf50af8](https://github.com/kilip/omed/commit/bf50af882281efdc9f46deaf3057a1c8af538b26))
* **api:** ✨ add mcpx fiber handler with bearer token verification ([7d62e02](https://github.com/kilip/omed/commit/7d62e02d041861893f7e44777394297a8cf823ca))
* **api:** ✨ add oauthx base helper for oauth2 drivers ([c4ca695](https://github.com/kilip/omed/commit/c4ca6958b049a28432fa3bbacb2641cbdfa7ec26))
* **api:** ✨ add oauthx driver interface and registry ([f5e99aa](https://github.com/kilip/omed/commit/f5e99aa126536971a3adb29cd99fd482531426fc))
* **api:** ✨ add personal access token schema and repository ([e1098d9](https://github.com/kilip/omed/commit/e1098d9583bee3d6e3798032d2ed7fbcfaf7aae6))
* **api:** ✨ add personal access token service ([584196a](https://github.com/kilip/omed/commit/584196a7ffd5043ce8dc8e47d7599c9632dc42ba))
* **api:** ✨ mount mcp endpoint with personal access token auth ([9dba581](https://github.com/kilip/omed/commit/9dba5813861773f5a5f5e9a9071f71e5e8d3e293))
* **api:** ✨ register fin insight tools with the mcp server ([2407f7b](https://github.com/kilip/omed/commit/2407f7b12f550be17c0a2a56359ee5fec80aeb3a))
* **deploy:** ✨ move helm chart to ./helm and require secret key ([0763708](https://github.com/kilip/omed/commit/076370884288a6e6d5d1aa5ca66ce43a85379400))
* **web:** ✨ add access tokens page ([f2d3faf](https://github.com/kilip/omed/commit/f2d3faf27b811a0f508ae935c5c508f1b775df61))
* **web:** ✨ add dedicated new and edit provider routes replacing drawer ([1fc9609](https://github.com/kilip/omed/commit/1fc960925dee9d6406c63059d903c4dbe3104213))
* **web:** ✨ add default base URLs, model AutoComplete and auto-fetch to ProviderForm ([28cd83e](https://github.com/kilip/omed/commit/28cd83e3a6aa135890558b37da1b2dfb5e899a2a))
* **web:** ✨ add provider models query hooks and translation keys ([593667f](https://github.com/kilip/omed/commit/593667fe6674e69a7be3d0f613b00d67389cc692))


### Bug Fixes

* **agent:** 🐛 address final review polish for validation and model state ([183c683](https://github.com/kilip/omed/commit/183c683b6c266301912715b94fe8d8dc9a78c9f3))
* **agent:** 🐛 resolve duplicate model id and ensure unique input selector ([1d07001](https://github.com/kilip/omed/commit/1d07001cac93aee5556abd3e4b6dcbf5754f9ce3))
* **api:** 🐛 fix account filter matching and verify truncation totals in insight service ([6350642](https://github.com/kilip/omed/commit/63506420cf236bfcdd4519f22279f97403431f93))
* **api:** 🐛 handle errcheck for close in mcpx handler tests ([d6af5ae](https://github.com/kilip/omed/commit/d6af5ae8b70c61401f87a44dfe98daa99f71f84c))
* **api:** 🐛 omit tool output schema in sdk to allow custom decimal serialization ([2eb50f5](https://github.com/kilip/omed/commit/2eb50f5f770408b1fc64a69e67b7b8bd8b1d0dbc))
* **api:** 🐛 return serialized call tool result to skip sdk schema mismatch ([c1a9f08](https://github.com/kilip/omed/commit/c1a9f08fa39271577b392e013ec70bae579f62dc))
* **api:** 🐛 update go.sum from ent code generation and check cs.Close in tests ([567959a](https://github.com/kilip/omed/commit/567959a70e5de0fb8f3e852503a7b0c299662cca))
* **e2e:** 🐛 update agent providers page object and regenerate OpenAPI contract ([c83adeb](https://github.com/kilip/omed/commit/c83adeb41a8cfcc717b5b117254aeecb905f93ac))

## [0.8.0](https://github.com/kilip/omed/compare/v0.7.0...v0.8.0) (2026-10-31)


### Features

* **agent:** ✨ add agent domain and repository ([ea1ed11](https://github.com/kilip/omed/commit/ea1ed1119accaa1e4a3e2123dfd5c65e599ee8ba))
* **agent:** ✨ add agent provider service ([241c5a0](https://github.com/kilip/omed/commit/241c5a0cbd40f10202eee7c95e2cd7a3357f0767))
* **agent:** ✨ add agent_providers ent schema and migration ([8b3dd56](https://github.com/kilip/omed/commit/8b3dd567aebba3d0a7ae05370e3c49b9251b0c7e))
* **agent:** ✨ add provider endpoints and integration tests ([431334e](https://github.com/kilip/omed/commit/431334ebce6049eca1887e30e7b1956b22412d71))
* **agent:** ✨ implement agent domain roles and provider repository ([019a488](https://github.com/kilip/omed/commit/019a4881b497f3b4948e9851fd218730355ee440))
* **api:** ✨ add agent config and chart values ([0385f64](https://github.com/kilip/omed/commit/0385f641e37068c1b2b7a1a9bee57adb3af8e4d5))
* **api:** ✨ add cashflow income and expense endpoints ([c5d8f4e](https://github.com/kilip/omed/commit/c5d8f4e4bdcc178e2963274cb95e4d51d6665099))
* **api:** ✨ add finance settings endpoints and openapi spec ([098dd5a](https://github.com/kilip/omed/commit/098dd5ac9daa91e4777318f10a1704721a4d72be))
* **api:** ✨ add item validation and draft editing to cashflow service ([2c9e140](https://github.com/kilip/omed/commit/2c9e1402384925cbf6fa1aa9bba1e09bbd5e8249))
* **api:** ✨ add llmx adapter contract and memory driver ([4b1f946](https://github.com/kilip/omed/commit/4b1f946532bbd6edaa77ae597adcae177bab3c6f))
* **api:** ✨ add llmx anthropic driver ([bb0aab0](https://github.com/kilip/omed/commit/bb0aab06431f937d80574e6d2cd5c41964c6dd34))
* **api:** ✨ add llmx gemini driver ([a94dc36](https://github.com/kilip/omed/commit/a94dc364b9567a4e51c9a576a54251e2980aac8a))
* **api:** ✨ add llmx openai driver ([bea8f8a](https://github.com/kilip/omed/commit/bea8f8aac7f7b41b33dd6a99388eed4fac3821e5))
* **api:** ✨ add llmx url guard ([4983228](https://github.com/kilip/omed/commit/4983228b2e2c466294d2619aa77277918a056c0c))
* **api:** ✨ add onboarding and decimal style to fin setting ([4188611](https://github.com/kilip/omed/commit/4188611fdaaa26b99cac1ada714a6a508efa6083))
* **api:** ✨ add secretx cipher ([477057f](https://github.com/kilip/omed/commit/477057fa779f0d4dc4fb8ed40cbd3f25700e8f71))
* **api:** ✨ expose income and expense get and update endpoints ([22f0864](https://github.com/kilip/omed/commit/22f0864111fbc524cf076c3170dab76ee4e2878f))
* **api:** ✨ store income and expense as multi-item documents ([7e967cf](https://github.com/kilip/omed/commit/7e967cfc666b84f19f48b775e42af2db253d0a97))
* **deploy:** ✨ add agent env to helm chart ([a722c15](https://github.com/kilip/omed/commit/a722c157c26f74627957af6959b69d384817914d))
* **web:** ✨ add agent module scaffolding ([21aacd5](https://github.com/kilip/omed/commit/21aacd515aa8f4df1bbed80e5e8fa28af7e53344))
* **web:** ✨ add agent providers screen ([b0885e3](https://github.com/kilip/omed/commit/b0885e3f538ff884435f72175ce6531ef567c1c3))
* **web:** ✨ add cashflow detail, edit, and status filter ([68c9548](https://github.com/kilip/omed/commit/68c95487874850f4d70648cfb88ad2c4a50d7c3b))
* **web:** ✨ add decimal style formatting and finance layout guard ([1ddc93b](https://github.com/kilip/omed/commit/1ddc93b6274c9eba99a5ec80ba1848d71fbf7b05))
* **web:** ✨ add fin launcher and income expense quick entry ([335e783](https://github.com/kilip/omed/commit/335e7836a64cdc31105c9631d86ea5be112569f7))
* **web:** ✨ add finance onboarding wizard and shared settings components ([8c52c8b](https://github.com/kilip/omed/commit/8c52c8bac315d72c396751181524429a22e7e139))
* **web:** ✨ add finance settings page and launcher tile ([157b308](https://github.com/kilip/omed/commit/157b3082467bbc0da0ec409985237ef557daa4ae))
* **web:** ✨ add money input and cashflow line helpers ([b177e78](https://github.com/kilip/omed/commit/b177e7875714fc04c54728b70581b4441b9eccc6))
* **web:** ✨ add multi-item cashflow form with posting and draft ([b600111](https://github.com/kilip/omed/commit/b600111150d5183a2136dfcc5cca762b4a5864e5))
* **web:** ✨ add route navigation loading overlay ([7ce310f](https://github.com/kilip/omed/commit/7ce310f6b149f54ae58114a850b72be322e39f39))


### Bug Fixes

* **agent:** 🐛 align provider service with llmx and base contracts ([26289cb](https://github.com/kilip/omed/commit/26289cb41918a7537b775ea306f9ca9f6eb13185))
* **agent:** 🐛 resolve whole-branch review findings ([7ed9ce1](https://github.com/kilip/omed/commit/7ed9ce1b037060f8188c141bc40516818635c534))
* **api:** 🐛 handle stream and body close errors ([deb4d50](https://github.com/kilip/omed/commit/deb4d50eac501eb2798e047bfd7ee969e7a8dbc8))
* **e2e:** 🐛 fix select option selector in income expense page ([9c8efb2](https://github.com/kilip/omed/commit/9c8efb2b84592d100ca4a57fc901f25848589237))
* **web:** 🐛 support 9-digit money input on opening balance ([5a6d828](https://github.com/kilip/omed/commit/5a6d828633ff44ae61a830c5ecf0c9cfd5708b7a))
* **web:** 🐛 surface non-field errors in provider drawer ([a4683a4](https://github.com/kilip/omed/commit/a4683a4aa5539950a6b4c05df884f4f5d729ceb3))

## [0.7.0](https://github.com/kilip/omed/compare/v0.6.0...v0.7.0) (2026-10-30)


### Features

* **deploy:** ✨ add chart icon ([c531479](https://github.com/kilip/omed/commit/c531479c1fc53fb43048c93be8a9209e9fca8890))
* **deploy:** ✨ add helm chart for api and web ([9f156a4](https://github.com/kilip/omed/commit/9f156a423b58b56b7b495d05249a2f7af807e4ae))
* **web:** ✨ load runtime config from config.js ([a4bea57](https://github.com/kilip/omed/commit/a4bea57b52dd3abd6c2eaabb2aa5f7ab12d59804))

## [0.6.0](https://github.com/kilip/omed/compare/v0.5.0...v0.6.0) (2026-10-27)


### ⚠ BREAKING CHANGES

* **api:** PostgreSQL tables now reside in module-scoped schemas (core, fin, cms) and runtime schema creation is disabled by default in favor of versioned Atlas migrations.
* **api:** amount, balance, total, and exchange rate fields in API models are now decimal strings instead of float64

### Features

* **api:** ✨ add multi-schema atlas database migrations ([492d501](https://github.com/kilip/omed/commit/492d50123f174d22fe1bd7b7d712d05fe2f56522))
* **api:** ✨ add quantity and unit price to postings ([7a99387](https://github.com/kilip/omed/commit/7a99387e98ddf0363069e687b6bc4411d07cbe83))
* **api:** ✨ add rate limiting, security headers and health endpoints ([48fced5](https://github.com/kilip/omed/commit/48fced58ee6ca7035900eb26a391c70ca9abaa41))
* **api:** ✨ represent money with shopspring decimal ([dae4824](https://github.com/kilip/omed/commit/dae482461bdfbf4426ba4d11ac2dbf214ef86f32))
* **web:** ✨ add quantity and unit price to postings editor ([12662df](https://github.com/kilip/omed/commit/12662dfe61d3a9c93861de81e3341833017728c4))
* **web:** ✨ handle decimal money strings ([a1a20fc](https://github.com/kilip/omed/commit/a1a20fcb27ca5994f6f201bc60501e505cff0eb9))


### Bug Fixes

* **api:** 🐛 fail closed when workspace context is missing ([6a18b72](https://github.com/kilip/omed/commit/6a18b726f14f3c17b6d994137f9e2e9dc50e5cfe))
* **api:** 🐛 fix errcheck and staticcheck lint issues in tests ([9f94be6](https://github.com/kilip/omed/commit/9f94be6d0f2c62020024dda6119eead7fa45dcae))
* **api:** 🐛 fix golangci-lint issues across api modules ([6816eee](https://github.com/kilip/omed/commit/6816eee23f6519db41753f93a3022b14560d4a11))
* **api:** 🐛 fix unchecked resp.body.close errcheck issues ([c82087c](https://github.com/kilip/omed/commit/c82087cbf5146df127cde5eb127e9c112b41ac20))
* **api:** 🐛 gate test route auth bypass behind the test flag ([0d92528](https://github.com/kilip/omed/commit/0d9252825494842cdea1472f0963a6c802f21445))
* **api:** 🐛 make entry post and void atomic ([a537378](https://github.com/kilip/omed/commit/a5373780fe0d9ba7d08c392bc7dc3e20e81bc95b))
* **api:** 🐛 make invitation accept atomic ([58cced2](https://github.com/kilip/omed/commit/58cced2c0e7ef8541d5b9dc157886f6820bf4121))
* **api:** 🐛 order shutdown and make port configurable ([e34d3da](https://github.com/kilip/omed/commit/e34d3da0d5d5aeaa6b0e686f61036a08c1f9d8b1))
* **api:** 🐛 return boot errors from run ([bff49ab](https://github.com/kilip/omed/commit/bff49ab3c5f96975803ff5fb5c5b123582ac69ff))
* **api:** 🐛 size body limit from the largest upload limit ([c028b4e](https://github.com/kilip/omed/commit/c028b4e283984f4d4159664cb3e10f28f307cca6))
* **api:** 🐛 sync ent tool checksums in go.sum ([6c0a79d](https://github.com/kilip/omed/commit/6c0a79d66f6f3f47be799431014c07eb5fab6b14))
* **api:** 🐛 validate status and period when reversing entries ([a4c7e15](https://github.com/kilip/omed/commit/a4c7e157ed862aff2cf4fb3adcb917fc0b2735a3))
* **auth:** 🐛 bind oauth state to browser with nonce cookie ([4339500](https://github.com/kilip/omed/commit/43395003f63ce865b43224fa1cfcedb980a49540))
* **auth:** 🐛 encrypt oauth provider tokens at rest ([ef8ad77](https://github.com/kilip/omed/commit/ef8ad77fed5cc1e43d5bac2390bcc035508a37c6))
* **auth:** 🐛 hash session tokens and stop returning them ([c18f0e6](https://github.com/kilip/omed/commit/c18f0e6f944cf88707cec06ba9345a10f7702845))
* **auth:** 🐛 require verified email before account linking ([0845a04](https://github.com/kilip/omed/commit/0845a043e81bbd4623d16cec7ac96affb53b5d8c))
* **auth:** 🐛 set secure flag on session cookie ([228e4a7](https://github.com/kilip/omed/commit/228e4a7288661b489e4be439ff901669a34a77bc))
* **auth:** 🐛 stop hardcoding email verified flag ([d5aa18e](https://github.com/kilip/omed/commit/d5aa18e62478d2b40f01217be7cd52b80b9d1915))
* **auth:** 🐛 validate auth secret and callback url ([bf1653d](https://github.com/kilip/omed/commit/bf1653d4b1f4328767859751d35d6e8755455802))
* **web:** 🐛 do not treat missing workspace as api error in shell ([684e134](https://github.com/kilip/omed/commit/684e1345821373ad39dc5172756c38500187a74e))
* **web:** 🐛 resolve antd deprecation warnings ([58d9520](https://github.com/kilip/omed/commit/58d9520c432c842131375248af4563ebebfebc99))
* **web:** 🐛 show api errors in the shell ([a94c6cd](https://github.com/kilip/omed/commit/a94c6cdc73712a25a6dadec7c12956707d386d24))
* **web:** 🐛 use design tokens in opening balance ([dbeb376](https://github.com/kilip/omed/commit/dbeb3767dde87936190e945b049e3a73298a50d7))

## [0.5.0](https://github.com/kilip/omed/compare/v0.4.0...v0.5.0) (2026-10-21)


### Features

* **api:** ✨ add cms ent schemas ([d1788be](https://github.com/kilip/omed/commit/d1788beffde8cdba4e16df8e3e9a3b65807b8ecd))
* **api:** ✨ add cms media config ([a53a34d](https://github.com/kilip/omed/commit/a53a34d4bebc397f9dc60f4ce8e6f3fc3ced6fc8))
* **api:** ✨ add cms media endpoints ([eea094a](https://github.com/kilip/omed/commit/eea094aaabe33d6b089852740cad62fe8aefaa6c))
* **api:** ✨ add cms media service ([20f8e00](https://github.com/kilip/omed/commit/20f8e009475f1658dd0989c4dede4a82ba9d81d9))
* **api:** ✨ add cms module skeleton and roles ([6e67659](https://github.com/kilip/omed/commit/6e6765932c0934a4f4f29337b3d1c40e9d491539))
* **api:** ✨ add cms page endpoints ([a69b53c](https://github.com/kilip/omed/commit/a69b53c6f7385b4b1a02c8e2493f1c70f9a01344))
* **api:** ✨ add cms page service ([24e70d6](https://github.com/kilip/omed/commit/24e70d697dbd23f3943b82a747a85f4514a6fd27))
* **api:** ✨ add cms post endpoints ([d9ac6d1](https://github.com/kilip/omed/commit/d9ac6d1f9a5b6f7059de935db32f8e93e933a10e))
* **api:** ✨ add cms post repository ([8e5aca8](https://github.com/kilip/omed/commit/8e5aca84c309bb1eb2bc6fdbbad30ddb4bb031e7))
* **api:** ✨ add cms post service ([baf3c55](https://github.com/kilip/omed/commit/baf3c5549d41758e567fcfca4132dfb9f8b22aa5))
* **api:** ✨ add cms public content endpoints ([8b2e597](https://github.com/kilip/omed/commit/8b2e597e957d9395e61704e0a279f6d2c0f02499))
* **api:** ✨ add cms publishing and slug helpers ([99cf8ac](https://github.com/kilip/omed/commit/99cf8ac7db5401669fdadd860a26f53ca5758bfe))
* **api:** ✨ add s3 storage, workspace buckets, and core_file ([79f8cde](https://github.com/kilip/omed/commit/79f8cde00c4aa0629729493ed48116d307ab4e94))
* **api:** ✨ add transaction attachments ([6e8097d](https://github.com/kilip/omed/commit/6e8097dbed6e9f0d34e5d3e970b4a410107fcc0e))
* **auth:** ✨ allow unauthenticated GET on /public routes ([138c9fd](https://github.com/kilip/omed/commit/138c9fd3857d84e915cf07de0ede6bb3cc06ac7d))
* **web:** ✨ add cms media library ([86e2896](https://github.com/kilip/omed/commit/86e2896a1b253d934e2de6a097ca27ad6c6becfd))
* **web:** ✨ add cms module scaffolding ([d1b8504](https://github.com/kilip/omed/commit/d1b85041f972a3b95ad807ed36495dc779bbbe44))
* **web:** ✨ add cms pages screens ([837a5bd](https://github.com/kilip/omed/commit/837a5bdd6bc6a8a9cd770bf0f9b1d85b5bfd316a))
* **web:** ✨ add cms posts screens ([ced9ff6](https://github.com/kilip/omed/commit/ced9ff6b852ebeff8e519cc42bae9cfe44eef4c3))
* **web:** ✨ add cms publish panel ([babf541](https://github.com/kilip/omed/commit/babf541378d61aa812973978d1357a9c8774ffd1))
* **web:** ✨ add transaction attachment picker and viewer ([475bc69](https://github.com/kilip/omed/commit/475bc698e7df5cb7fc62fab06c52e034e843f849))


### Bug Fixes

* **web:** 🐛 include react-markdown in optimizeDeps ([f49ebf0](https://github.com/kilip/omed/commit/f49ebf0cf05bda7fd159f16165227ac0b3ceefcd))

## [0.4.0](https://github.com/kilip/omed/compare/v0.3.0...v0.4.0) (2026-10-20)


### Features

* **api:** ✨ add accounting period management ([e8e9981](https://github.com/kilip/omed/commit/e8e9981c524e088d9ffb270a27d52598eaaf5691))
* **api:** ✨ add coa template preview and personal expense accounts ([d461c54](https://github.com/kilip/omed/commit/d461c54a9239255f829d2733049eec9819dc315b))
* **api:** ✨ add cursor pagination and query filters to fin search ([68a8a88](https://github.com/kilip/omed/commit/68a8a88205e035a7fb9aa38d6809529d8268a21e))
* **api:** ✨ add ent schema for transactions, entries, and postings ([f0a9e9c](https://github.com/kilip/omed/commit/f0a9e9c3a6475cca8ffd497b4d2f5df244886d0e))
* **api:** ✨ add financial summary report endpoint ([1ecc56e](https://github.com/kilip/omed/commit/1ecc56e59b22ba57f4d6bf6cdc1115b58aa31d77))
* **api:** ✨ add transaction and entry controllers with openapi and tests ([37e007a](https://github.com/kilip/omed/commit/37e007a081b8a80e69e6d37c6af9541ada47ad6e))
* **api:** ✨ add transaction and entry domain, models, and authz ([b07db3c](https://github.com/kilip/omed/commit/b07db3cdfe8548fdb59812e7787fab656a2508f5))
* **api:** ✨ add user locale and currency preferences ([d2d144e](https://github.com/kilip/omed/commit/d2d144e35b815a06c215bb41be30453f57a5227c))
* **api:** ✨ implement fx rounding auto-balancing, coa templates, and multi-currency ledger balances ([951e10e](https://github.com/kilip/omed/commit/951e10e8442467ef2c027e29539a40517b8efb7a))
* **api:** ✨ implement multi-currency money helper, exchange rates, and base currency setting ([7297fe9](https://github.com/kilip/omed/commit/7297fe9befbd76a4dad5c376c7a459c93eeb16a6))
* **api:** ✨ implement period revaluation, realized fx, and trial balance report ([5939d42](https://github.com/kilip/omed/commit/5939d424714a81fa7f95a4e00201e2ec04b68ce4))
* **api:** ✨ implement transaction and entry repositories with numbering ([cd94f9f](https://github.com/kilip/omed/commit/cd94f9f7f34777e5a90d772eec555a8f0015e5e6))
* **api:** ✨ implement transaction and entry services with validation and period close guard ([2d6d4ba](https://github.com/kilip/omed/commit/2d6d4ba745661c23e65028aaea1f0025b4f5426e))
* **web:** ✨ add coa preview in setup and opening balance page ([f22be3c](https://github.com/kilip/omed/commit/f22be3c26c99326706879004a48396fa4a4a43f2))
* **web:** ✨ add create entry drawer ([4b77feb](https://github.com/kilip/omed/commit/4b77febde9c53384ef608325589e04ee8bbb05c9))
* **web:** ✨ add create transaction drawer ([d1be788](https://github.com/kilip/omed/commit/d1be788cf06cb53c97826c2ceb71a70643b304e9))
* **web:** ✨ add currency-aware money formatting ([83eaef4](https://github.com/kilip/omed/commit/83eaef45b883a88b05245d7615bdd5e489344cac))
* **web:** ✨ add cursor-aware unwrapPage helper ([5d37b33](https://github.com/kilip/omed/commit/5d37b330948cab6455f7824f22d64147158cb13d))
* **web:** ✨ add dedicated create and update transaction routes ([3497039](https://github.com/kilip/omed/commit/349703924d5fecc3f52fd03c5664fc39ad2b9837))
* **web:** ✨ add entry detail view with post, edit and delete ([cbe3fb2](https://github.com/kilip/omed/commit/cbe3fb2f81934a62055f6e596e92d7d83a4157fd))
* **web:** ✨ add entry reverse and void actions ([94c4e9c](https://github.com/kilip/omed/commit/94c4e9c1f1a380a5fb1badd6ce9c9ae1586db30c))
* **web:** ✨ add fin period management and setup configuration ([2f9ddee](https://github.com/kilip/omed/commit/2f9ddee203fb7cd2cda3e5ad3700461a235fa676))
* **web:** ✨ add fin query options for transactions and entries ([609a4c1](https://github.com/kilip/omed/commit/609a4c112b6a62f87992291adc635ee99f75e5f0))
* **web:** ✨ add fin select and status components ([2c3ae29](https://github.com/kilip/omed/commit/2c3ae291ec4c528618e1b383b17f6588fbe5fd30))
* **web:** ✨ add hydrate fallback to shell ([ecd195b](https://github.com/kilip/omed/commit/ecd195b2ce283249a2820233a4acdf8b522f761b))
* **web:** ✨ add postings editor ([770db8f](https://github.com/kilip/omed/commit/770db8ff816b311fe51324dfa5b599ceb2c6f913))
* **web:** ✨ add transaction detail ([65b4072](https://github.com/kilip/omed/commit/65b4072ddb827f223dcd1dae316dc1b52f9f0d2b))
* **web:** ✨ add transactions list ([0bc3e6d](https://github.com/kilip/omed/commit/0bc3e6dac319e2c7e4f9529c84d8f50a3dd89f77))
* **web:** ✨ connect dashboard to real api and remove mocks ([d68a53a](https://github.com/kilip/omed/commit/d68a53ad6353a08fe32c74a475684bb4ae699bc3))
* **web:** ✨ gate fin actions by role ([48bdf83](https://github.com/kilip/omed/commit/48bdf83c4071e92d5bdb568fca693fb46b7b042a))
* **web:** ✨ group finance navigation under accounting menu ([018fb49](https://github.com/kilip/omed/commit/018fb49d47db8836a3c7253d1e6ce11395941e34))
* **web:** ✨ integrate trial balance endpoint and remove mocks ([5f058e4](https://github.com/kilip/omed/commit/5f058e495026a038dffa8955beb4d991d2d6131f))
* **web:** ✨ link accounts and entries to ledger ([8b967c2](https://github.com/kilip/omed/commit/8b967c23756a2abd71874af05edde661125eefdc))
* **web:** ✨ register fin journal detail route ([b95b2f2](https://github.com/kilip/omed/commit/b95b2f208b905d6f7d53c2f3e847ede86d6ca96b))
* **web:** ✨ register fin transactions route and menu ([b6c99dc](https://github.com/kilip/omed/commit/b6c99dc86edafb0a0f9a74fa4965e15213b55ddf))
* **web:** ✨ replace entry drawer with dedicated new entry page ([c691f8c](https://github.com/kilip/omed/commit/c691f8cf39d5754103344afa028bdf5e4c0fc50b))
* **web:** ✨ sync language preference with user locale ([2401f06](https://github.com/kilip/omed/commit/2401f0660cc44243ed5f4dafe909e07729c299df))
* **web:** ✨ translate fin api error codes ([752a539](https://github.com/kilip/omed/commit/752a539c8a424293ac5ac797427b51d2bbc20387))
* **web:** ✨ wire general ledger to api ([0a1f2ed](https://github.com/kilip/omed/commit/0a1f2edfcaeb03eabfe634ecbb7f8b11df21686a))
* **web:** ✨ wire journal entries list to api ([1adf433](https://github.com/kilip/omed/commit/1adf4334017fb10fc16d31088c844e4ef8439fa5))


### Bug Fixes

* **api:** 🐛 fallback line descriptions and support single role in testctl ([9ac0c15](https://github.com/kilip/omed/commit/9ac0c159bfab9dc88cbe6bd23d9cc3404ba07d79))
* **auth:** 🐛 resolve social login hang and payload casing ([e813cca](https://github.com/kilip/omed/commit/e813ccac408222ed5bcc234b13317c2f2d532e9b))
* **e2e:** 🐛 remove duplicate period declaration in permissions steps ([02667a8](https://github.com/kilip/omed/commit/02667a8ab3640ce7ed049287d23f49d8b79c8b76))
* **web:** 🐛 add period defaults and prevent e2e bundle reloads ([70e3518](https://github.com/kilip/omed/commit/70e35187d7642ca4c94d99ba73732738732b9973))
* **web:** 🐛 build image using bun monorepo context ([14d0fec](https://github.com/kilip/omed/commit/14d0fec23c496adc3f46676e0a6f540b6c6ad85f))

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
