# B-HIS 运行状态（H01）

## 当前总状态（2026-11-01 08:31 Asia/Shanghai）

- 本机：http://127.0.0.1:8888/login；0.3.11-draft
- REG-PATIENT-SELECT-01：**有条件完成**（H06 R5）；R1=d12b761；REG-UI-FOLD-01=a87328e 已关；REG-DOC-01=4cf1d5e 已关；仍欠 INACTIVE 样本 / gitlab 推送
- 刚闭合：UI90-04B-LEFT-03=**done** 关闭 tip **d6b711f**（H05 实现 **f3558a8**；H06 抽验 **d6b711f** 建议通过；**明确不签 90%**；未 SCORE；origin 关片前 ahead≈3 待普通 push）；UI90-04B-LEFT-02=**done** tip **8695f2a**（H05 **ef46286**；H06 **44d330d**）；UI90-04B-LEFT-01=**done** tip **ea80717**（H05 **de00f3e**；H06 **61b28b4**）；UI90-04C-FIX-04=**done** tip **c3c5483**（H05 **4b2e605**；H06 **64ec746**）；UI90-04C-FIX-03=**done** tip **9443941**（H05 **39a62e0**；H06 **afad64d**）；UI90-04C-FIX-02=**done** tip **f0aa482**；UI90-04C-REVERIFY-01=**done** tip **5c829e8**；UI90-04A-FIX-01=**done** tip **be229e4**（H06 抽验 tip **2b69533**）；UI90-04A-EVID-01=**done** tip **593e7ef**；DEBT-MOCK-CHAIN-ALIGN-01=**done** tip **8a71ee4**；P1-THIN-SMOKE-01=**done** tip **42f2d09**；P1-ORDER-RECENT-01=**done** tip **42f2d09**；P1-DIAG-VERSION-DIFF-01=**done** tip **e8073b7**；P1-ORDER-COMMON-01=**done** tip **120a409**；P1-DIAG-ICD-FIND-01=**done** tip **4243b19**；P1-DIAG-COMMON-01=**done** tip **929deaf**；P1-DIAG-RESUME-01=**done** tip **5fa6b06**（**P1 三刀薄包 done**）；P1-ORDER-FIND-01=**done** tip **8865146**；P1-PRINT-STATUS-01=**done** tip **c2589d7**；P0-E-INTEGRATION-01/**Eb-H18**=**通过** tip **649cb13**（**整片 P0-E-INTEGRATION-01=done**；**P0/M1=done**）
- 条件：H06 抽验建议通过已读；看板关片；未改业务源码（关片仅进度文件）；未签 UI90 90%；未 SCORE
- 上一刀：UI90-04B-LEFT-03=d6b711f（H05=f3558a8；H06=d6b711f 建议通过；不签90%；未SCORE）
- 下一刀 / current：**idle**
- 说明：关片节点；UserOperationsLogPanel 本页壳宽 1008（--uol-shell-w；panelW=1008；resultsLeft≈1；stackH=126）已落盘并测量抽验通过；LEFT-01/02 回归过；未改 AppShell/MasterPanelHeader；证据 deliverables/h05/ui90-04b-left-03-2026-11-01.md + deliverables/h06/ui90-04b-left-03-spotcheck-2026-11-01.md；**明确不签 90%**；SCORE / 04A-SCORE / 04C-SCORE 后置；表列后置 LEFT-04；AppShell/458→后置 03b
- 共享反馈原语：useSubmitFeedback + SubmitStatusBanner **已稳**（Register→Dispense；五专项 harness 全绿；禁平行栈）
- 看板：done 79 / pending 16 / total 95（83%）；current **idle**
- origin：codex/register-slot-card-20261028；关片 tip=**d6b711f**（全 **d6b711fa01a59af446831408941b217d08eba14f**，含实现 f3558a8 与 H06 抽验）；关片前 **ahead≈3** 待关片 docs 提交后普通 push（禁 force/reset）
- 已知债：gitlab 192.168.1.138:8929 不通；openapi 整文件入 d12b761；contracts README tip 漂移；大量无关 dirty（禁 reset／禁混交）；完整官方 ICD / 主数据 Dict / 强制编码 / 后端检索**另立卡**；模板密度/AUTOSAVE 未做；H05b=否；mock-chain vs D5 同键冲突债**已清理**；UI90-04A **FIX 已关但仍不签 90%**；UI90-04C **FIX-02/03/04 已关但仍不签 90%**；04A-SCORE / 04C-SCORE 后置另卡；04B-LEFT-01/02/03 已关仍不签 90%；表列→LEFT-04；AppShell→03b
- 禁项：DIAG-AUTOSAVE；官方 ICD/Dict/强制编码/新 search API/022；看板标 90% done；夹带 dirty；force/reset；公网部署；本片未跑 sync_progress.py；**关片仅看板+STATUS+bak，未改业务源码、未 force/reset**
