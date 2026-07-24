#!/usr/bin/env python3
"""Structural contract tests for Skills primary orchestrator package.

Drives real shipped files under skills/ (prompts, wrappers, install scripts).
Run from repo root or skills/: python skills/tests/test_skills_orchestrator.py
"""

from __future__ import annotations

import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import unittest
from hashlib import sha256
from pathlib import Path

SKILLS_ROOT = Path(__file__).resolve().parents[1]
PROMPTS = SKILLS_ROOT / "prompts"
COPILOT_PROMPTS = SKILLS_ROOT / "install" / "copilot" / "prompts"
CLAUDE_GOVERN_SKILL = (
    SKILLS_ROOT / "install" / "claude" / "skills" / "govern" / "SKILL.md"
)
CLAUDE_AUDIT_SKILL = (
    SKILLS_ROOT / "install" / "claude" / "skills" / "audit" / "SKILL.md"
)
GROK_GOVERN_SKILL = SKILLS_ROOT / "install" / "grok" / "skills" / "govern" / "SKILL.md"
GROK_AUDIT_SKILL = SKILLS_ROOT / "install" / "grok" / "skills" / "audit" / "SKILL.md"
INSTALL_SH = SKILLS_ROOT / "install.sh"
INSTALL_PS1 = SKILLS_ROOT / "install.ps1"
INSTALL_PS1_ISOLATED = SKILLS_ROOT / "tests" / "test_install_ps1_isolated.ps1"
README = SKILLS_ROOT / "README.md"
CORE_TEMPLATES = SKILLS_ROOT.parent / "docs" / "templates" / "goal-folder"
SKILLS_TEMPLATES = SKILLS_ROOT / "templates" / "goal-folder"
CORE_WORKSPACE_TEMPLATE = SKILLS_ROOT.parent / "docs" / "templates" / "workspace-context.md"
SKILLS_WORKSPACE_TEMPLATE = SKILLS_ROOT / "templates" / "workspace-context.md"
CORE_CONTRACTS = SKILLS_ROOT.parent / "docs" / "contracts"
SKILLS_CONTRACTS = SKILLS_ROOT / "contracts"
CORE_PRINCIPLES = SKILLS_ROOT.parent / "docs" / "architecture" / "principles.md"
CORE_AGENTS = SKILLS_ROOT.parent / "AGENTS.md"
CONTRACT_SCHEMA_ID = (
    "https://github.com/magicvr/goal-governance/schema/skills-consumer-contract/v1"
)
MATRIX_SCHEMA_ID = (
    "https://github.com/magicvr/goal-governance/schema/"
    "skills-consumer-compatibility-matrix/v1"
)
RUNTIME_EVIDENCE_SCHEMA_ID = (
    "https://github.com/magicvr/goal-governance/schema/runtime-evidence/v1"
)


def normalized_sha256(path: Path) -> str:
    """Hash repository text with checkout line endings normalized to LF."""
    payload = path.read_bytes().replace(b"\r\n", b"\n")
    return sha256(payload).hexdigest().upper()


CONTRACT_MIRROR_FILES = (
    "skills-consumer-contract.schema.json",
    "skills-consumer-contract.json",
    "skills-consumer-compatibility-matrix.schema.json",
    "skills-consumer-compatibility-matrix.json",
    "runtime-evidence.schema.json",
    "fixtures/valid/manifest-0.1.0.json",
    "fixtures/valid/declared-adapter-0.1.0.json",
    "fixtures/invalid/missing-contract-schema-id.json",
    "fixtures/invalid/missing-support-baseline.json",
    "fixtures/invalid/declared-adapter-without-commitment.json",
    "fixtures/invalid/reversed-protocol-range.json",
    "fixtures/invalid/unstable-cross-minor-range.json",
    "fixtures/invalid/unsupported-protocol-0.2.0.json",
    "fixtures/invalid/fabricated-predecessor-0.0.0.json",
)
SEMVER_RE = re.compile(
    r"^(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)"
    r"(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$"
)


class TestSkillsOrchestratorPackage(unittest.TestCase):
    def test_primary_orchestrator_file_exists(self) -> None:
        path = PROMPTS / "00-govern-orchestrator.md"
        self.assertTrue(path.is_file(), f"missing primary orchestrator: {path}")

    def test_orchestrator_encodes_lifecycle_and_classification(self) -> None:
        text = (PROMPTS / "00-govern-orchestrator.md").read_text(encoding="utf-8")
        # Lifecycle language
        self.assertIn("设立目标", text)
        self.assertIn("推进", text)
        self.assertRegex(text, r"审计|关门")
        self.assertRegex(text, r"信息发现|信息就绪")
        self.assertIn("I-00N", text)
        self.assertRegex(text, r"最晚需要阶段|信息门禁")
        # Classification / scan
        self.assertIn("goal-tree", text)
        self.assertRegex(text, r"S0|情境|分类")
        self.assertRegex(text, r"未关门|总目的")
        self.assertRegex(text, r"scaffold|工作区骨架")
        self.assertRegex(text, r"不完整安装|同级必备")
        # Confirm before write + primitives
        self.assertRegex(text, r"确认")
        for name in (
            "01-create-new-goal",
            "02-record-decision",
            "03-update-execution",
            "04-write-audit",
        ):
            self.assertIn(name, text)
        # Primary role marker
        self.assertRegex(text, r"主入口|primary|单一")
        # Defaults / confirm-with-user (positive framing)
        self.assertRegex(text, r"仓库根")
        self.assertRegex(text, r"默认策略|待确认|问用户")
        self.assertRegex(text, r"web/")  # as optional project convention example
        self.assertRegex(text, r"完成标准|硬约束")
        # GOAL-005 phase B: opinion ledger + user gates
        self.assertRegex(text, r"意见台账|P-004|开放必改")
        self.assertRegex(text, r"independent|交叉")
        self.assertRegex(text, r"S4|审计响应")
        self.assertIn("05-independent-audit", text)
        self.assertRegex(text, r"P-005|未知项")
        for marker in (
            "信息就绪台账",
            "到期 required 信息项",
            "不机械创建两个信息子目标",
            "accepted-residual",
            "non-blocking",
            "deferred",
            "contracts/skills-consumer-contract.json",
        ):
            self.assertIn(marker, text)

    def test_primitives_exist_and_marked(self) -> None:
        for fname in (
            "01-create-new-goal.md",
            "02-record-decision.md",
            "03-update-execution.md",
            "04-write-audit.md",
        ):
            path = PROMPTS / fname
            self.assertTrue(path.is_file(), f"missing primitive: {path}")
            body = path.read_text(encoding="utf-8")
            self.assertRegex(
                body,
                r"primitive|原语",
                msg=f"{fname} should label itself as primitive",
            )
        audit04 = (PROMPTS / "04-write-audit.md").read_text(encoding="utf-8")
        self.assertRegex(audit04, r"source")
        self.assertRegex(audit04, r"verdict")
        self.assertRegex(audit04, r"independent|self")
        for fname in (
            "01-create-new-goal.md",
            "02-record-decision.md",
            "03-update-execution.md",
            "04-write-audit.md",
            "05-independent-audit.md",
        ):
            text = (PROMPTS / fname).read_text(encoding="utf-8")
            self.assertRegex(
                text,
                r"P-005|信息项|信息需求|I-00N",
                msg=f"{fname} must consume the information-readiness protocol",
            )

    def test_independent_audit_prompt_exists(self) -> None:
        path = PROMPTS / "05-independent-audit.md"
        self.assertTrue(path.is_file(), f"missing independent audit core: {path}")
        text = path.read_text(encoding="utf-8")
        self.assertIn("independent", text)
        self.assertRegex(text, r"03-audit")
        self.assertRegex(text, r"status|progress")
        self.assertRegex(text, r"govern|/govern")

    def test_prompts_readme_primary_vs_primitive(self) -> None:
        text = (PROMPTS / "README.md").read_text(encoding="utf-8")
        self.assertIn("00-govern-orchestrator", text)
        self.assertRegex(text, r"primary|主入口")
        self.assertRegex(text, r"primitive|原语")

    def test_core_templates_are_mirrored_by_skills_package(self) -> None:
        """The Skills package must distribute the canonical core templates unchanged."""
        if not CORE_TEMPLATES.is_dir():
            self.skipTest(
                "canonical docs/templates layer is not present in a standalone Skills copy"
            )
        self.assertTrue(
            CORE_TEMPLATES.is_dir(), f"missing canonical templates: {CORE_TEMPLATES}"
        )
        self.assertTrue(
            SKILLS_TEMPLATES.is_dir(),
            f"missing Skills template mirror: {SKILLS_TEMPLATES}",
        )
        for name in ("00-meta.md", "01-decision.md", "02-execution.md", "03-audit.md"):
            canonical = CORE_TEMPLATES / name
            mirror = SKILLS_TEMPLATES / name
            self.assertTrue(canonical.is_file(), f"missing canonical template: {canonical}")
            self.assertTrue(mirror.is_file(), f"missing Skills template mirror: {mirror}")
            self.assertEqual(
                canonical.read_bytes(),
                mirror.read_bytes(),
                f"template mirror drift: {canonical} != {mirror}",
            )
        self.assertTrue(
            CORE_WORKSPACE_TEMPLATE.is_file(),
            f"missing canonical workspace template: {CORE_WORKSPACE_TEMPLATE}",
        )
        self.assertTrue(
            SKILLS_WORKSPACE_TEMPLATE.is_file(),
            f"missing Skills workspace template mirror: {SKILLS_WORKSPACE_TEMPLATE}",
        )
        self.assertEqual(
            CORE_WORKSPACE_TEMPLATE.read_bytes(),
            SKILLS_WORKSPACE_TEMPLATE.read_bytes(),
            "workspace context template mirror drift",
        )
        workspace_template = CORE_WORKSPACE_TEMPLATE.read_text(encoding="utf-8")
        for marker in (
            "root_goal",
            "canonical_scope",
            "shared_materials_catalog",
            "reference_id",
            "workspace_id",
            "sha256",
        ):
            self.assertIn(marker, workspace_template)
        self.assertIn(
            "信息就绪与未知项",
            (CORE_TEMPLATES / "00-meta.md").read_text(encoding="utf-8"),
        )
        self.assertIn(
            "信息需求与阶段门禁",
            (CORE_TEMPLATES / "01-decision.md").read_text(encoding="utf-8"),
        )
        self.assertIn(
            "信息就绪核对",
            (CORE_TEMPLATES / "03-audit.md").read_text(encoding="utf-8"),
        )
        docs_readme = CORE_TEMPLATES.parent / "README.md"
        skills_readme = SKILLS_ROOT / "README.md"
        self.assertIn("canonical", docs_readme.read_text(encoding="utf-8").lower())
        self.assertIn("分发镜像", skills_readme.read_text(encoding="utf-8"))

    @staticmethod
    def _load_json(path: Path) -> dict[str, object]:
        with path.open(encoding="utf-8") as handle:
            value = json.load(handle)
        if not isinstance(value, dict):
            raise AssertionError(f"contract JSON root must be an object: {path}")
        return value

    def _semver_core(self, value: object, label: str) -> tuple[int, int, int]:
        self.assertIsInstance(value, str, msg=f"{label} must be a string")
        match = SEMVER_RE.fullmatch(value)
        self.assertIsNotNone(match, msg=f"{label} is not a SemVer value: {value!r}")
        assert match is not None
        return tuple(int(match.group(name)) for name in ("major", "minor", "patch"))

    def _assert_semver_range(self, value: object, label: str) -> tuple[tuple[int, int, int], tuple[int, int, int]]:
        self.assertIsInstance(value, dict, msg=f"{label} must be an object")
        assert isinstance(value, dict)
        self.assertEqual(set(value), {"minInclusive", "maxExclusive"}, msg=label)
        minimum = self._semver_core(value["minInclusive"], f"{label}.minInclusive")
        maximum = self._semver_core(value["maxExclusive"], f"{label}.maxExclusive")
        self.assertLess(minimum, maximum, msg=f"{label} must not be reversed")
        if minimum[0] == 0:
            self.assertEqual(maximum[0], 0, msg=f"{label} must not cross 0.x major")
            self.assertEqual(
                maximum[1],
                minimum[1] + 1,
                msg=f"{label} must stay within one unstable 0.y line",
            )
            self.assertEqual(maximum[2], 0, msg=f"{label} must end at the next minor")
        return minimum, maximum

    def _assert_contract_instance(self, payload: dict[str, object], schema: dict[str, object]) -> None:
        self.assertEqual(
            set(payload),
            {
                "contractSchemaId",
                "contractFormat",
                "contractFormatVersion",
                "canonical",
                "protocol",
                "supportBaseline",
                "templateSet",
                "adapterCompatibilityStatus",
                "adapters",
            },
        )
        self.assertEqual(payload["contractSchemaId"], schema["$id"])
        self.assertEqual(payload["contractFormat"], "goal-governance.skills-consumer-contract")
        self._semver_core(payload["contractFormatVersion"], "contractFormatVersion")

        canonical = payload["canonical"]
        self.assertIsInstance(canonical, dict)
        assert isinstance(canonical, dict)
        self.assertEqual(
            canonical,
            {
                "owner": "docs/contracts",
                "manifestPath": "docs/contracts/skills-consumer-contract.json",
                "schemaPath": "docs/contracts/skills-consumer-contract.schema.json",
            },
        )

        protocol = payload["protocol"]
        self.assertIsInstance(protocol, dict)
        assert isinstance(protocol, dict)
        self.assertEqual(set(protocol), {"version", "versionPolicy", "publicContract"})
        protocol_version = self._semver_core(protocol["version"], "protocol.version")
        self.assertEqual(protocol["versionPolicy"], "semver-2.0.0")
        public_contract = protocol["publicContract"]
        self.assertIsInstance(public_contract, dict)
        assert isinstance(public_contract, dict)
        self.assertEqual(
            public_contract,
            {
                "goalTemplateFiles": [
                    "00-meta.md",
                    "01-decision.md",
                    "02-execution.md",
                    "03-audit.md",
                ],
                "requiredFrontmatter": [
                    "status",
                    "created",
                    "updated",
                    "parent",
                    "version",
                ],
                "hostEntrypoints": ["govern", "audit"],
            },
        )

        support_baseline = payload["supportBaseline"]
        self.assertIsInstance(support_baseline, dict)
        assert isinstance(support_baseline, dict)
        self.assertEqual(
            set(support_baseline),
            {"firstSupportedProtocol", "previousSupportedProtocol"},
        )
        first_supported = self._semver_core(
            support_baseline["firstSupportedProtocol"],
            "supportBaseline.firstSupportedProtocol",
        )
        self.assertLessEqual(first_supported, protocol_version)
        previous_supported = support_baseline["previousSupportedProtocol"]
        if previous_supported is not None:
            previous = self._semver_core(
                previous_supported,
                "supportBaseline.previousSupportedProtocol",
            )
            self.assertLess(previous, first_supported)

        template_set = payload["templateSet"]
        self.assertIsInstance(template_set, dict)
        assert isinstance(template_set, dict)
        self.assertEqual(
            set(template_set),
            {"version", "implementsProtocol", "canonicalPath", "mirrorPath"},
        )
        self._semver_core(template_set["version"], "templateSet.version")
        minimum, maximum = self._assert_semver_range(
            template_set["implementsProtocol"], "templateSet.implementsProtocol"
        )
        self.assertLessEqual(minimum, protocol_version)
        self.assertLess(protocol_version, maximum)
        self.assertEqual(template_set["canonicalPath"], "docs/templates/goal-folder")
        self.assertEqual(template_set["mirrorPath"], "skills/templates/goal-folder")

        status = payload["adapterCompatibilityStatus"]
        self.assertIn(status, {"I-002-pending", "declared", "verified"})
        adapters = payload["adapters"]
        self.assertIsInstance(adapters, list)
        assert isinstance(adapters, list)
        if status == "I-002-pending":
            self.assertEqual(adapters, [])
        else:
            self.assertTrue(adapters, msg=f"{status} requires declared adapters")
        adapter_ids = [adapter.get("id") for adapter in adapters if isinstance(adapter, dict)]
        self.assertEqual(len(adapter_ids), len(set(adapter_ids)), msg="adapter ids must be unique")
        for adapter in adapters:
            self.assertIsInstance(adapter, dict)
            assert isinstance(adapter, dict)
            self.assertEqual(
                set(adapter),
                {
                    "id",
                    "supportsProtocol",
                    "entrypoints",
                    "supportCommitment",
                    "verificationStatus",
                },
            )
            self.assertRegex(str(adapter["id"]), r"^[a-z0-9][a-z0-9-]*$")
            adapter_minimum, adapter_maximum = self._assert_semver_range(
                adapter["supportsProtocol"], f"adapter {adapter['id']} supportsProtocol"
            )
            self.assertLessEqual(adapter_minimum, protocol_version)
            self.assertLess(protocol_version, adapter_maximum)
            self.assertTrue(
                set(adapter["entrypoints"]).issubset(
                    set(public_contract["hostEntrypoints"])
                )
            )
            self.assertIn(adapter["supportCommitment"], {"declared", "committed"})
            self.assertIn(adapter["verificationStatus"], {"unverified", "verified"})
        if status == "verified":
            self.assertTrue(
                all(adapter["verificationStatus"] == "verified" for adapter in adapters),
                msg="verified status requires every declared adapter to be verified",
            )

    def test_core_contracts_are_mirrored_by_skills_package(self) -> None:
        """The distributed contract must remain a byte-for-byte mirror of canonical docs."""
        if not CORE_CONTRACTS.is_dir():
            self.skipTest("canonical docs/contracts layer is not present in a standalone Skills copy")
        self.assertTrue(SKILLS_CONTRACTS.is_dir(), f"missing Skills contract mirror: {SKILLS_CONTRACTS}")
        for relative_name in CONTRACT_MIRROR_FILES:
            canonical = CORE_CONTRACTS / relative_name
            mirror = SKILLS_CONTRACTS / relative_name
            self.assertTrue(canonical.is_file(), f"missing canonical contract: {canonical}")
            self.assertTrue(mirror.is_file(), f"missing Skills contract mirror: {mirror}")
            self.assertEqual(
                canonical.read_bytes(),
                mirror.read_bytes(),
                f"contract mirror drift: {canonical} != {mirror}",
            )

    def test_contract_schema_manifest_and_fixtures_enforce_d002_semantics(self) -> None:
        """Validate D-002/D-003 schema identity, fields, and boundary fixtures."""
        schema = self._load_json(CORE_CONTRACTS / "skills-consumer-contract.schema.json")
        self.assertEqual(schema["$schema"], "https://json-schema.org/draft/2020-12/schema")
        self.assertEqual(schema["$id"], CONTRACT_SCHEMA_ID)
        self.assertFalse(schema["additionalProperties"])
        self.assertEqual(
            set(schema["required"]),
            {
                "contractSchemaId",
                "contractFormat",
                "contractFormatVersion",
                "canonical",
                "protocol",
                "supportBaseline",
                "templateSet",
                "adapterCompatibilityStatus",
                "adapters",
            },
        )
        properties = schema["properties"]
        self.assertEqual(properties["contractSchemaId"]["const"], CONTRACT_SCHEMA_ID)
        self.assertEqual(
            properties["protocol"]["properties"]["versionPolicy"]["const"],
            "semver-2.0.0",
        )
        self.assertEqual(
            schema["allOf"][0]["then"]["properties"]["adapters"]["maxItems"],
            0,
        )
        self.assertEqual(
            set(schema["properties"]["supportBaseline"]["required"]),
            {"firstSupportedProtocol", "previousSupportedProtocol"},
        )
        self.assertEqual(
            set(schema["$defs"]["adapter"]["required"]),
            {
                "id",
                "supportsProtocol",
                "entrypoints",
                "supportCommitment",
                "verificationStatus",
            },
        )

        self._assert_contract_instance(
            self._load_json(CORE_CONTRACTS / "skills-consumer-contract.json"), schema
        )
        for name in ("manifest-0.1.0.json", "declared-adapter-0.1.0.json"):
            self._assert_contract_instance(
                self._load_json(CORE_CONTRACTS / "fixtures" / "valid" / name), schema
            )
        for name in (
            "missing-contract-schema-id.json",
            "missing-support-baseline.json",
            "declared-adapter-without-commitment.json",
            "reversed-protocol-range.json",
            "unstable-cross-minor-range.json",
            "unsupported-protocol-0.2.0.json",
        ):
            with self.assertRaises(AssertionError, msg=name):
                self._assert_contract_instance(
                    self._load_json(CORE_CONTRACTS / "fixtures" / "invalid" / name),
                    schema,
                )

    def test_d003_declares_baseline_and_tiered_adapter_scope(self) -> None:
        """D-003 keeps commitment separate from version-fixed runtime evidence."""
        manifest = self._load_json(CORE_CONTRACTS / "skills-consumer-contract.json")
        self.assertEqual(manifest["contractFormatVersion"], "0.2.0")
        self.assertEqual(
            manifest["supportBaseline"],
            {
                "firstSupportedProtocol": "0.1.0",
                "previousSupportedProtocol": None,
            },
        )
        self.assertEqual(manifest["adapterCompatibilityStatus"], "declared")
        adapters = manifest["adapters"]
        self.assertIsInstance(adapters, list)
        assert isinstance(adapters, list)
        by_id = {adapter["id"]: adapter for adapter in adapters}
        self.assertEqual(len(by_id), len(adapters), msg="adapter ids must be unique")
        self.assertEqual(
            set(by_id),
            {"claude-code-cli", "grok-build-cli", "github-copilot-cli"},
        )
        self.assertEqual(by_id["claude-code-cli"]["supportCommitment"], "committed")
        self.assertEqual(
            by_id["github-copilot-cli"]["supportCommitment"], "committed"
        )
        self.assertEqual(by_id["grok-build-cli"]["supportCommitment"], "committed")
        for adapter in by_id.values():
            self.assertEqual(
                adapter["supportsProtocol"],
                {"minInclusive": "0.1.0", "maxExclusive": "0.2.0"},
            )
            self.assertEqual(adapter["entrypoints"], ["govern", "audit"])
        self.assertEqual(by_id["claude-code-cli"]["verificationStatus"], "verified")
        self.assertEqual(by_id["grok-build-cli"]["verificationStatus"], "verified")
        self.assertEqual(by_id["github-copilot-cli"]["verificationStatus"], "verified")
        self.assertNotIn("web-readonly-parser", by_id)

    def test_candidate_compatibility_matrix_keeps_runtime_coverage_explicit(self) -> None:
        """Verified host cells and remaining candidate gaps must stay distinct."""
        schema = self._load_json(
            CORE_CONTRACTS / "skills-consumer-compatibility-matrix.schema.json"
        )
        runtime_schema = self._load_json(
            CORE_CONTRACTS / "runtime-evidence.schema.json"
        )
        matrix = self._load_json(
            CORE_CONTRACTS / "skills-consumer-compatibility-matrix.json"
        )
        manifest = self._load_json(CORE_CONTRACTS / "skills-consumer-contract.json")
        self.assertEqual(schema["$id"], MATRIX_SCHEMA_ID)
        self.assertEqual(runtime_schema["$id"], RUNTIME_EVIDENCE_SCHEMA_ID)
        self.assertEqual(matrix["schemaId"], MATRIX_SCHEMA_ID)
        self.assertEqual(matrix["format"], "goal-governance.skills-consumer-compatibility-matrix")
        self.assertEqual(matrix["candidateRevision"], "v0.9.0")
        self.assertEqual(matrix["canonicalContractPath"], "docs/contracts/skills-consumer-contract.json")
        self.assertEqual(matrix["protocol"]["current"], manifest["protocol"]["version"])
        self.assertIsNone(matrix["protocol"]["previous"])
        self.assertEqual(
            matrix["protocol"]["previousStatus"],
            "not-applicable-first-supported-protocol",
        )
        self.assertEqual(matrix["requiredEntrypoints"], ["govern", "audit"])
        negative = {item["id"]: item for item in matrix["negativeFixtures"]}
        self.assertIn("unsupported-protocol-0.2.0", negative)
        self.assertIn("no-fabricated-predecessor", negative)
        for item in negative.values():
            self.assertTrue(
                (SKILLS_ROOT.parent / item["path"]).is_file(),
                msg=item["path"],
            )
        self.assertEqual(negative["unsupported-protocol-0.2.0"]["kind"], "unsupported-protocol")
        self.assertEqual(negative["no-fabricated-predecessor"]["kind"], "fabricated-predecessor")
        fabricated = self._load_json(SKILLS_ROOT.parent / negative["no-fabricated-predecessor"]["path"])
        self.assertEqual(fabricated["protocol"], manifest["protocol"])
        self.assertEqual(fabricated["supportBaseline"]["previousSupportedProtocol"], "0.0.0")

        consumers = {item["id"]: item for item in matrix["consumers"]}
        self.assertEqual(
            set(consumers),
            {
                "claude-code-cli",
                "grok-build-cli",
                "github-copilot-cli",
                "web-readonly-parser",
            },
        )
        self.assertEqual(consumers["claude-code-cli"]["host"]["version"], "2.1.218")
        self.assertEqual(consumers["grok-build-cli"]["host"]["version"], "0.2.111")
        self.assertEqual(consumers["github-copilot-cli"]["host"]["version"], "1.0.71")
        self.assertEqual(consumers["github-copilot-cli"]["host"]["product"], "GitHub Copilot CLI")
        adapters_by_id = {adapter["id"]: adapter for adapter in manifest["adapters"]}
        # Claude + Grok + Copilot: all six entrypoints runtime-verified 2026-07-24
        for consumer_id in (
            "claude-code-cli",
            "grok-build-cli",
            "github-copilot-cli",
        ):
            entrypoints = {
                entry["name"]: entry for entry in consumers[consumer_id]["entrypoints"]
            }
            self.assertEqual(
                consumers[consumer_id]["contractVerificationStatus"],
                adapters_by_id[consumer_id]["verificationStatus"],
            )
            self.assertEqual(set(entrypoints), {"govern", "audit"})
            for name in ("govern", "audit"):
                self.assertEqual(entrypoints[name]["status"], "runtime-verified")
                self.assertTrue(entrypoints[name]["evidence"])
                for path in entrypoints[name]["evidence"]:
                    self.assertTrue((SKILLS_ROOT.parent / path).is_file(), msg=path)
                    self.assertIn("2026-07-24", path)
        web = consumers["web-readonly-parser"]
        self.assertEqual(web["kind"], "goal-document-parser")
        self.assertEqual(web["supportCommitment"], "not-applicable")
        self.assertEqual(web["contractVerificationStatus"], "not-applicable")
        self.assertEqual(web["entrypoints"][0]["status"], "automated-verified")

    def test_p005_core_contract_guards_unknown_information_gates(self) -> None:
        """Keep P-005's actual gates from regressing to a keyword-only policy."""
        if not (CORE_PRINCIPLES.is_file() and CORE_AGENTS.is_file()):
            self.skipTest("core methodology is not present in a standalone Skills copy")

        principles = CORE_PRINCIPLES.read_text(encoding="utf-8")
        p005 = principles.split("## P-005", 1)[1].split(
            "## 原则与落地文档对照", 1
        )[0]
        for marker in (
            "### 信息需求登记",
            "### 设立与阶段门禁",
            "### 残余风险与用户裁决",
            "### 子目标拆分",
            "目标可以创建为 `draft` 或 `active`，即使信息表仍有开放项",
            "不得伪造完整方案",
            "| 级别 |",
            "required",
            "non-blocking",
            "| 影响门禁 |",
            "| 最晚需要阶段 |",
            "| 验证 / 收集动作 |",
            "| 状态 |",
            "| 延期 / 复核 |",
            "| 证据 / 结论 |",
            "规划门禁",
            "实施门禁",
            "关门门禁",
            "有界实验只能进入其明确的**信息收集范围**",
            "不等同于验证通过",
            "不得放行实验范围之外的实施",
            "暂停受影响范围、记录事实，并回流到信息表、决策或路线图",
            "编排器不得静默推断",
            "适用期限",
            "缓解/监控方式",
            "不等同于 `verified`",
            "`deferred required` 自动按开放 required 处理并阻断受影响门禁",
            "不是每个目标的固定两个子目标",
            "独立范围、依赖、产物证据、持续时间或并行价值",
            "低风险、可逆",
        ):
            self.assertIn(marker, p005, msg=f"P-005 semantic contract lost: {marker}")

        agents = CORE_AGENTS.read_text(encoding="utf-8")
        for marker in (
            "允许带未知立项",
            "信息需求登记",
            "阶段门禁",
            "发现后的回流",
            "按规模拆分",
            "禁止为每个低风险问题机械创建两个子目标",
        ):
            self.assertIn(marker, agents, msg=f"AGENTS P-005 summary drifted: {marker}")

    def test_p005_operational_contract_guards_prompts_and_templates(self) -> None:
        """Prompts/templates must operationalize gates, not merely name P-005."""
        orchestrator = (PROMPTS / "00-govern-orchestrator.md").read_text(encoding="utf-8")
        for marker in (
            "没有表时，不假定“没有未知”",
            "到达最晚需要阶段的 `deferred required`",
            "有到期 required 信息项时，停止自动放行",
            "有界实验只允许其明确收集范围，I-00N 保持 `collecting`",
            "不因“以后再收集”自动创建两个子目标",
            "`accepted-residual` 有用户书面接受、范围和复审触发",
        ):
            self.assertIn(
                marker,
                orchestrator,
                msg=f"orchestrator P-005 gate lost: {marker}",
            )

        meta = (SKILLS_TEMPLATES / "00-meta.md").read_text(encoding="utf-8")
        decision = (SKILLS_TEMPLATES / "01-decision.md").read_text(encoding="utf-8")
        execution = (SKILLS_TEMPLATES / "02-execution.md").read_text(encoding="utf-8")
        audit = (SKILLS_TEMPLATES / "03-audit.md").read_text(encoding="utf-8")
        self.assertIn("P-005 不要求设立目标时已经知道一切", meta)
        self.assertIn("required / non-blocking", meta)
        self.assertIn("<deferred 时填理由、责任人、复核触发>", meta)
        self.assertIn("必须指向用户的书面决策或审计响应", decision)
        self.assertIn("不等同于 `verified`", decision)
        self.assertIn(
            "不能把 `open`、`deferred` 或 `accepted-residual` 写成已验证事实",
            execution,
        )
        self.assertIn("未关闭的 required 信息项应作为 finding", audit)

        audit_prompt = (PROMPTS / "04-write-audit.md").read_text(encoding="utf-8")
        self.assertIn(
            "若 required 信息项已到期、影响 scope、或 `accepted-residual` 没有用户书面接受，应作为 finding",
            audit_prompt,
        )

    def test_copilot_govern_wrapper_is_primary(self) -> None:
        path = COPILOT_PROMPTS / "govern.md"
        self.assertTrue(path.is_file(), f"missing primary wrapper: {path}")
        text = path.read_text(encoding="utf-8")
        self.assertIn("/govern", text)
        self.assertIn("00-govern-orchestrator", text)
        self.assertRegex(text, r"primary|主入口")

    def test_install_scripts_install_govern(self) -> None:
        sh = INSTALL_SH.read_text(encoding="utf-8")
        ps1 = INSTALL_PS1.read_text(encoding="utf-8")
        self.assertRegex(sh, r"govern")
        self.assertRegex(ps1, r"govern")
        self.assertRegex(sh, r"00-govern-orchestrator|/govern")
        self.assertRegex(ps1, r"00-govern-orchestrator|/govern")

    def test_core_d004_mirror_is_complete(self) -> None:
        """GOAL-019 D-004: skills/core ships methodology subset without tech-stack."""
        core = SKILLS_ROOT / "core" / "docs"
        required = (
            core / "README.md",
            core / "architecture" / "principles.md",
            core / "architecture" / "workspace-protocol.md",
            core / "architecture" / "overview.md",
            core / "architecture" / "directory-layout.md",
            core / "templates" / "README.md",
            core / "templates" / "workspace-context.md",
            core / "templates" / "goal-folder" / "00-meta.md",
            core / "templates" / "goal-folder" / "01-decision.md",
            core / "templates" / "goal-folder" / "02-execution.md",
            core / "templates" / "goal-folder" / "03-audit.md",
        )
        for path in required:
            self.assertTrue(path.is_file(), f"missing core mirror file: {path}")
        self.assertFalse(
            (core / "architecture" / "tech-stack.md").is_file(),
            "core mirror must not include tech-stack.md",
        )
        principles = (core / "architecture" / "principles.md").read_text(encoding="utf-8")
        for marker in ("P-001", "P-002", "P-003", "P-004", "P-005"):
            self.assertIn(marker, principles)
        layout = (core / "architecture" / "directory-layout.md").read_text(encoding="utf-8")
        self.assertIn("workspace-", layout)
        self.assertNotIn("goal-governance/web", layout.replace("\\", "/"))
        # goal-folder five-pack should match package templates mirror
        for name in ("00-meta.md", "01-decision.md", "02-execution.md", "03-audit.md"):
            self.assertEqual(
                (core / "templates" / "goal-folder" / name).read_bytes(),
                (SKILLS_TEMPLATES / name).read_bytes(),
                f"core templates drift from skills/templates: {name}",
            )
        sh = INSTALL_SH.read_text(encoding="utf-8")
        ps1 = INSTALL_PS1.read_text(encoding="utf-8")
        self.assertIn("install_core_docs", sh)
        self.assertIn("Install-CoreDocs", ps1)
        self.assertIn("core/docs", sh.replace("\\", "/"))
        self.assertIn("core", ps1)
        self.assertRegex(sh, r"workspace-|init-workspace")
        self.assertRegex(ps1, r"workspace-|InitWorkspace|init-workspace")
        self.assertNotIn(r"docs\goals\goal-tree", ps1)
        # GOAL-019 phase C: optional workspace scaffold (explicit slugs)
        self.assertIn("--init-workspace", sh)
        self.assertIn("init_workspace_skeleton", sh)
        self.assertIn("--workspace-slug", sh)
        self.assertIn("--root-slug", sh)
        self.assertRegex(ps1, r"InitWorkspace|init-workspace")
        self.assertIn("Initialize-WorkspaceSkeleton", ps1)
        self.assertRegex(ps1, r"WorkspaceSlug|workspace-slug")
        self.assertRegex(ps1, r"RootSlug|root-slug")
        self.assertRegex(sh, r"no silent default|D-005")
        self.assertRegex(ps1, r"no silent default|D-005")
        # GOAL-019 A-001 F-002: refuse overwrite when workspace path exists
        self.assertRegex(sh, r"already exists|refuse overwrite")
        self.assertRegex(ps1, r"already exists|refuse overwrite")

    @unittest.skipUnless(sys.platform.startswith("win"), "InitWorkspace refuse smoke is Windows/ps1-first")
    def test_init_workspace_refuses_existing_path(self) -> None:
        """GOAL-019 A-001 F-002: second -InitWorkspace on same path must fail."""
        pwsh = shutil.which("powershell") or shutil.which("pwsh")
        if not pwsh:
            self.skipTest("PowerShell not found on PATH")
        with tempfile.TemporaryDirectory(prefix="gg-init-refuse-") as tmp:
            target = Path(tmp)
            skills = SKILLS_ROOT
            cmd_base = [
                pwsh,
                "-NoProfile",
                "-NonInteractive",
                "-ExecutionPolicy",
                "Bypass",
                "-File",
                str(INSTALL_PS1),
                "-InitWorkspace",
                "-WorkspaceSlug",
                "refuse-demo",
                "-RootSlug",
                "refuse-root",
                "-SkillsDir",
                str(skills),
            ]
            first = subprocess.run(
                cmd_base,
                cwd=str(target),
                capture_output=True,
                text=True,
                encoding="utf-8",
                errors="replace",
                check=False,
            )
            self.assertEqual(
                first.returncode,
                0,
                msg=f"first init failed:\n{first.stdout}\n{first.stderr}",
            )
            ws = target / "docs" / "workspace-001-refuse-demo" / "workspace.md"
            self.assertTrue(ws.is_file(), msg="first init did not create workspace.md")
            second = subprocess.run(
                cmd_base,
                cwd=str(target),
                capture_output=True,
                text=True,
                encoding="utf-8",
                errors="replace",
                check=False,
            )
            self.assertNotEqual(
                second.returncode,
                0,
                msg=f"second init should refuse:\n{second.stdout}\n{second.stderr}",
            )
            combined = (second.stdout or "") + (second.stderr or "")
            self.assertRegex(
                combined,
                r"(?i)already exists|refuse",
                msg=f"refuse message missing:\n{combined}",
            )

    def test_monorepo_agents_architecture_not_optional_supplement(self) -> None:
        """GOAL-019 A-001 F-001: root AGENTS must not call architecture optional."""
        agents = (SKILLS_ROOT.parent / "AGENTS.md").read_text(encoding="utf-8")
        self.assertNotRegex(agents, r"architecture 原则全文可选补充")
        self.assertNotRegex(agents, r"architecture.*可选补充")
        self.assertRegex(agents, r"同级必备|原则全文\*\*必备\*\*|必备.*同级")

    def test_install_all_ships_contract_mirror(self) -> None:
        sh = INSTALL_SH.read_text(encoding="utf-8")
        ps1 = INSTALL_PS1.read_text(encoding="utf-8")
        self.assertIn("CONTRACTS_SRC", sh)
        self.assertIn('"$SKILLS_DIR/contracts"', sh)
        self.assertIn("$ContractsSrc", ps1)
        self.assertIn("'contracts'", ps1)

    def test_install_default_slash_is_govern_and_audit_opt_in_primitives(self) -> None:
        """Default install: /govern + /audit; form-fill primitives stay opt-in."""
        sh = INSTALL_SH.read_text(encoding="utf-8")
        ps1 = INSTALL_PS1.read_text(encoding="utf-8")
        self.assertIn("--with-primitives", sh)
        self.assertRegex(ps1, r"WithPrimitives|with-primitives")
        self.assertIn("WRAPPER_NAMES=(govern audit)", sh)
        self.assertIn("$wrapperNames = @('govern', 'audit')", ps1)
        self.assertIn("INSTALL_PRIMITIVE_WRAPPERS", sh)
        self.assertIn("$WithPrimitives", ps1)
        self.assertIn("new-goal", sh)
        self.assertIn("new-goal", ps1)
        self.assertRegex(sh, r"skills/audit|audit/SKILL")
        self.assertRegex(ps1, r"skills\\audit|audit\\SKILL")
        self.assertRegex(sh, r"INSTALL_PRIMITIVE_WRAPPERS.*1|with-primitives")

    def test_agents_template_does_not_force_web_app_dir(self) -> None:
        text = (SKILLS_ROOT / "AGENTS.template.md").read_text(encoding="utf-8")
        self.assertIn("代码与文档边界", text)
        self.assertRegex(text, r"仓库根")
        self.assertRegex(text, r"默认策略|问用户|待确认")
        self.assertRegex(text, r"web/")
        self.assertRegex(text, r"正确做法|硬约束")
        self.assertNotRegex(
            text,
            r"应用代码仅在 `\{\{APP_DIR\}\}`",
            msg="template must not force APP_DIR-only application code",
        )

    def test_information_readiness_is_shipped_to_all_rule_surfaces(self) -> None:
        """P-005 must survive the reusable rule and host-install surfaces."""
        paths = (
            SKILLS_ROOT / "AGENTS.template.md",
            SKILLS_ROOT / "install" / "claude" / "AGENTS.md",
            SKILLS_ROOT / "install" / "copilot" / "copilot-instructions.md",
            CLAUDE_GOVERN_SKILL,
            GROK_GOVERN_SKILL,
            COPILOT_PROMPTS / "govern.md",
            CLAUDE_AUDIT_SKILL,
            GROK_AUDIT_SKILL,
            COPILOT_PROMPTS / "audit.md",
        )
        for path in paths:
            text = path.read_text(encoding="utf-8")
            self.assertRegex(
                text,
                r"P-005|信息就绪|I-00N",
                msg=f"missing P-005 information-readiness contract: {path}",
            )

        advanced = (
            COPILOT_PROMPTS / "new-goal.md",
            COPILOT_PROMPTS / "log-decision.md",
            COPILOT_PROMPTS / "update-execution.md",
            COPILOT_PROMPTS / "write-audit.md",
        )
        for path in advanced:
            text = path.read_text(encoding="utf-8")
            self.assertRegex(
                text,
                r"P-005|信息就绪|I-00N",
                msg=f"advanced Copilot primitive drifted from P-005: {path}",
            )

    def test_workspace_protocol_is_shipped_to_skills_surfaces(self) -> None:
        """Workspace isolation and fixed-reference rules must not remain core-doc-only."""
        protocol = (
            SKILLS_ROOT.parent / "docs" / "architecture" / "workspace-protocol.md"
        ).read_text(encoding="utf-8")
        for marker in (
            "隐式单工作区",
            "canonical_scope",
            "sha256",
            "fail-closed",
            "串行子目标",
        ):
            self.assertIn(marker, protocol)

        core_prompts = tuple(PROMPTS / name for name in (
            "00-govern-orchestrator.md",
            "01-create-new-goal.md",
            "02-record-decision.md",
            "03-update-execution.md",
            "04-write-audit.md",
            "05-independent-audit.md",
        ))
        for path in core_prompts:
            text = path.read_text(encoding="utf-8")
            self.assertIn(
                "docs/workspace-<NNN>-<slug>/workspace.md",
                text,
                msg=f"missing workspace scan: {path}",
            )
            self.assertRegex(
                text,
                r"Root Goal|root_goal|canonical 范围|canonical_scope",
                msg=f"missing workspace binding: {path}",
            )

        rule_surfaces = (
            SKILLS_ROOT / "AGENTS.template.md",
            SKILLS_ROOT / "install" / "claude" / "AGENTS.md",
            SKILLS_ROOT / "install" / "copilot" / "copilot-instructions.md",
            CLAUDE_GOVERN_SKILL,
            GROK_GOVERN_SKILL,
            COPILOT_PROMPTS / "govern.md",
            CLAUDE_AUDIT_SKILL,
            GROK_AUDIT_SKILL,
            COPILOT_PROMPTS / "audit.md",
        )
        for path in rule_surfaces:
            text = path.read_text(encoding="utf-8")
            self.assertIn(
                "docs/workspace-<NNN>-<slug>/workspace.md",
                text,
                msg=f"missing workspace rule: {path}",
            )
            self.assertRegex(
                text,
                r"隐式单工作区|fail closed|工作区上下文",
                msg=f"missing fail-closed compatibility behavior: {path}",
            )

    def test_docs_readme_hash_ledger_matches_template_bytes(self) -> None:
        """The published canonical/mirror ledger must describe the shipped bytes."""
        readme = (SKILLS_ROOT.parent / "docs" / "README.md").read_text(encoding="utf-8")
        for name in ("00-meta.md", "01-decision.md", "02-execution.md", "03-audit.md"):
            canonical = CORE_TEMPLATES / name
            mirror = SKILLS_TEMPLATES / name
            digest = normalized_sha256(canonical)
            self.assertEqual(digest, normalized_sha256(mirror))
            self.assertIn(name, readme)
            self.assertIn(digest, readme)
        workspace_digest = normalized_sha256(CORE_WORKSPACE_TEMPLATE)
        self.assertEqual(workspace_digest, normalized_sha256(SKILLS_WORKSPACE_TEMPLATE))
        self.assertIn("workspace-context.md", readme)
        self.assertIn(workspace_digest, readme)
        for name in (
            "skills-consumer-contract.schema.json",
            "skills-consumer-contract.json",
            "skills-consumer-compatibility-matrix.schema.json",
            "skills-consumer-compatibility-matrix.json",
            "runtime-evidence.schema.json",
        ):
            canonical = CORE_CONTRACTS / name
            mirror = SKILLS_CONTRACTS / name
            digest = normalized_sha256(canonical)
            self.assertEqual(digest, normalized_sha256(mirror))
            self.assertIn(name, readme)
            self.assertIn(digest, readme)

    def test_portability_skills_pkg_and_required_architecture(self) -> None:
        """Package may rename skills dir; architecture is co-required (GOAL-019)."""
        template = (SKILLS_ROOT / "AGENTS.template.md").read_text(encoding="utf-8")
        orch = (PROMPTS / "00-govern-orchestrator.md").read_text(encoding="utf-8")
        govern = (COPILOT_PROMPTS / "govern.md").read_text(encoding="utf-8")
        create01 = (PROMPTS / "01-create-new-goal.md").read_text(encoding="utf-8")
        for text, label in (
            (template, "AGENTS.template"),
            (orch, "orchestrator"),
            (govern, "govern wrapper"),
        ):
            self.assertRegex(
                text,
                r"SKILLS_PKG|改名|也可改名|其他名字|其他名",
                msg=f"{label} should allow renamed skills package",
            )
        # Architecture is required for complete install — not optional product framing
        self.assertRegex(template, r"同级必备|必备")
        self.assertIn("docs/architecture/principles.md", template)
        self.assertRegex(orch, r"不完整安装|同级必备")
        self.assertIn("docs/architecture/principles.md", orch)
        self.assertRegex(govern, r"不完整安装|同级必备|principles")
        # S0 scaffold + user-confirmed slug (phase B)
        self.assertRegex(orch, r"scaffold|工作区骨架")
        self.assertRegex(orch, r"用户确认")
        self.assertRegex(create01, r"步骤 0|工作区骨架|scaffold")
        self.assertRegex(create01, r"禁止静默|用户确认")
        self.assertNotIn("GOAL-001-main-vision", orch)
        self.assertNotIn("GOAL-001-main-vision", create01)
        self.assertIn("SKILLS_PKG", orch)
        # Prefer positive defaults over long ban-lists in the fenced body
        ban_hits = len(__import__("re").findall(r"^[-*]\s*禁止", orch, flags=__import__("re").M))
        self.assertLessEqual(ban_hits, 4, "orchestrator should not be a ban-list prompt")

    def test_skills_readme_documents_primary_and_audit_paths(self) -> None:
        text = README.read_text(encoding="utf-8")
        self.assertIn("/govern", text)
        self.assertIn("/audit", text)
        self.assertIn("00-govern-orchestrator", text)
        self.assertIn("05-independent-audit", text)
        self.assertRegex(text, r"primary|主入口")
        self.assertRegex(text, r"primitive|原语|advanced")
        self.assertRegex(text, r"with-primitives|WithPrimitives")
        self.assertRegex(text, r"Claude|\.claude")
        self.assertRegex(text, r"Grok|\.grok")
        self.assertIn("SKILL.md", text)
        self.assertIn("contracts/", text)
        self.assertIn("core/", text)
        self.assertRegex(text, r"同级必备|不完整安装")
        self.assertIn("principles", text)

    def test_skills_readme_default_install_documents_govern_and_audit(self) -> None:
        """F-017 guard: README manual/script sections must match default govern+audit surface."""
        text = README.read_text(encoding="utf-8")
        norm = text.replace("\\", "/")
        # Manual install: both skills/paths for each host family
        self.assertIn(".claude/skills/audit", norm)
        self.assertIn(".grok/skills/audit", norm)
        self.assertRegex(text, r"audit\.prompt\.md|prompts/audit")
        self.assertIn("skills/audit/SKILL.md", norm)
        # Must not revive the old "Copilot = only govern prompt" claim
        self.assertNotIn("仅** govern prompt", text)
        self.assertNotIn("**仅** govern prompt", text)
        self.assertNotRegex(text, r"(?i)--copilot[^\n]{0,80}仅\s*govern\s*prompt")
        # Explicit default surface language
        self.assertRegex(text, r"`/govern`\s*\+\s*`/audit`")
        self.assertRegex(text, r"--claude[^\n]*audit", re.I)
        self.assertRegex(text, r"--grok[^\n]*audit", re.I)
        self.assertRegex(text, r"--copilot[^\n]*(audit|默认双入口)", re.I)

    def _assert_primary_govern_skill(self, path: Path, host_label: str) -> None:
        self.assertTrue(path.is_file(), f"missing {host_label} skill: {path}")
        text = path.read_text(encoding="utf-8")
        self.assertRegex(text, r"(?m)^name:\s*govern\s*$")
        self.assertRegex(text, r"description:")
        self.assertIn("00-govern-orchestrator", text)
        self.assertRegex(text, r"SKILLS_PKG|prompts/00-govern")
        self.assertRegex(text, r"设立目标|set-goal|推进|lifecycle|生命周期")
        self.assertRegex(text, r"primary|主入口|单一")
        # Must not present four form ops as the default product surface
        self.assertNotRegex(
            text,
            r"(?i)default.*(new-goal|log-decision|四选一|填表菜单)",
        )

    def test_claude_govern_skill_source(self) -> None:
        self._assert_primary_govern_skill(CLAUDE_GOVERN_SKILL, "Claude Code")

    def test_grok_govern_skill_source(self) -> None:
        self._assert_primary_govern_skill(GROK_GOVERN_SKILL, "Grok Build")

    def _assert_audit_skill(self, path: Path, host_label: str) -> None:
        self.assertTrue(path.is_file(), f"missing {host_label} audit skill: {path}")
        text = path.read_text(encoding="utf-8")
        self.assertRegex(text, r"(?m)^name:\s*audit\s*$")
        self.assertIn("05-independent-audit", text)
        self.assertRegex(text, r"independent|交叉")
        self.assertRegex(text, r"status|progress")

    def test_claude_audit_skill_source(self) -> None:
        self._assert_audit_skill(CLAUDE_AUDIT_SKILL, "Claude Code")

    def test_grok_audit_skill_source(self) -> None:
        self._assert_audit_skill(GROK_AUDIT_SKILL, "Grok Build")

    def test_copilot_audit_wrapper(self) -> None:
        path = COPILOT_PROMPTS / "audit.md"
        self.assertTrue(path.is_file(), f"missing audit wrapper: {path}")
        text = path.read_text(encoding="utf-8")
        self.assertIn("/audit", text)
        self.assertIn("05-independent-audit", text)
        self.assertRegex(text, r"independent|交叉")

    def test_install_scripts_wire_claude_and_grok_skills(self) -> None:
        sh = INSTALL_SH.read_text(encoding="utf-8")
        ps1 = INSTALL_PS1.read_text(encoding="utf-8")
        for text, label in ((sh, "install.sh"), (ps1, "install.ps1")):
            self.assertRegex(text, r"--grok|-Grok", msg=f"{label} needs grok flag")
            self.assertIn(".claude/skills/govern", text.replace("\\", "/"))
            self.assertIn(".grok/skills/govern", text.replace("\\", "/"))
            self.assertIn("audit", text)
            self.assertIn("SKILL.md", text)
            self.assertRegex(text, r"00-govern-orchestrator")
            self.assertRegex(text, r"05-independent-audit")

    def test_install_ps1_isolated_smoke_script_exists(self) -> None:
        self.assertTrue(
            INSTALL_PS1_ISOLATED.is_file(),
            f"missing F-018 isolated install script: {INSTALL_PS1_ISOLATED}",
        )
        text = INSTALL_PS1_ISOLATED.read_text(encoding="utf-8")
        self.assertIn("install.ps1", text)
        self.assertIn("audit", text)
        self.assertIn("govern", text)
        self.assertIn("contracts", text)

    @unittest.skipUnless(sys.platform.startswith("win"), "F-018 PS1 install smoke is Windows-first")
    def test_install_ps1_isolated_all_produces_govern_and_audit(self) -> None:
        """Execute install.ps1 -All in a temp project; assert default product surface."""
        pwsh = shutil.which("powershell") or shutil.which("pwsh")
        if not pwsh:
            self.skipTest("PowerShell not found on PATH")

        self.assertTrue(INSTALL_PS1.is_file(), f"missing {INSTALL_PS1}")
        with tempfile.TemporaryDirectory(prefix="gg-skills-install-") as tmp:
            target = Path(tmp)
            skills_dest = target / "skills"
            cmd = [
                pwsh,
                "-NoProfile",
                "-NonInteractive",
                "-ExecutionPolicy",
                "Bypass",
                "-File",
                str(INSTALL_PS1),
                "-All",
                "-SkillsDir",
                str(skills_dest),
            ]
            proc = subprocess.run(
                cmd,
                cwd=str(target),
                capture_output=True,
                text=True,
                encoding="utf-8",
                errors="replace",
                timeout=180,
                env={**os.environ, "TERM": "dumb"},
            )
            combined = (proc.stdout or "") + "\n" + (proc.stderr or "")
            self.assertEqual(
                proc.returncode,
                0,
                msg=f"install.ps1 -All failed (code={proc.returncode}):\n{combined}",
            )

            required = [
                target / "AGENTS.md",
                target / ".claude" / "skills" / "govern" / "SKILL.md",
                target / ".claude" / "skills" / "audit" / "SKILL.md",
                target / ".grok" / "skills" / "govern" / "SKILL.md",
                target / ".grok" / "skills" / "audit" / "SKILL.md",
                target / ".github" / "copilot-instructions.md",
                target / ".github" / "prompts" / "govern.prompt.md",
                target / ".github" / "prompts" / "audit.prompt.md",
                skills_dest / "prompts" / "00-govern-orchestrator.md",
                skills_dest / "prompts" / "05-independent-audit.md",
                skills_dest / "contracts" / "skills-consumer-contract.schema.json",
                skills_dest / "contracts" / "skills-consumer-contract.json",
                skills_dest / "contracts" / "skills-consumer-compatibility-matrix.schema.json",
                skills_dest / "contracts" / "skills-consumer-compatibility-matrix.json",
                skills_dest / "contracts" / "runtime-evidence.schema.json",
            ]
            missing = [str(p) for p in required if not p.is_file()]
            self.assertEqual(missing, [], msg=f"missing install outputs: {missing}")

            # Default install must NOT ship form-fill advanced slashes
            self.assertFalse(
                (target / ".github" / "prompts" / "new-goal.prompt.md").is_file(),
                "new-goal.prompt.md must not install without -WithPrimitives",
            )

            govern = (target / ".claude" / "skills" / "govern" / "SKILL.md").read_text(
                encoding="utf-8"
            )
            audit = (target / ".claude" / "skills" / "audit" / "SKILL.md").read_text(
                encoding="utf-8"
            )
            self.assertIn("00-govern-orchestrator", govern)
            self.assertIn("05-independent-audit", audit)
            for name in (
                "skills-consumer-contract.schema.json",
                "skills-consumer-contract.json",
                "skills-consumer-compatibility-matrix.schema.json",
                "skills-consumer-compatibility-matrix.json",
                "runtime-evidence.schema.json",
            ):
                self.assertEqual(
                    (skills_dest / "contracts" / name).read_bytes(),
                    (SKILLS_CONTRACTS / name).read_bytes(),
                    msg=f"installed contract drift: {name}",
                )

    @unittest.skipUnless(sys.platform.startswith("win"), "advanced install smoke is Windows-first")
    def test_install_ps1_with_primitives_keeps_information_readiness(self) -> None:
        """Optional Copilot primitives must not silently fall behind P-005."""
        pwsh = shutil.which("powershell") or shutil.which("pwsh")
        if not pwsh:
            self.skipTest("PowerShell not found on PATH")

        with tempfile.TemporaryDirectory(prefix="gg-skills-primitives-") as tmp:
            target = Path(tmp)
            cmd = [
                pwsh,
                "-NoProfile",
                "-NonInteractive",
                "-ExecutionPolicy",
                "Bypass",
                "-File",
                str(INSTALL_PS1),
                "-All",
                "-WithPrimitives",
                "-SkillsDir",
                str(target / "skills"),
            ]
            proc = subprocess.run(
                cmd,
                cwd=str(target),
                capture_output=True,
                text=True,
                encoding="utf-8",
                errors="replace",
                timeout=180,
                env={**os.environ, "TERM": "dumb"},
            )
            combined = (proc.stdout or "") + "\n" + (proc.stderr or "")
            self.assertEqual(proc.returncode, 0, msg=combined)
            for name in (
                "new-goal.prompt.md",
                "log-decision.prompt.md",
                "update-execution.prompt.md",
                "write-audit.prompt.md",
            ):
                text = (target / ".github" / "prompts" / name).read_text(
                    encoding="utf-8"
                )
                self.assertRegex(text, r"P-005|信息就绪|I-00N", msg=name)


def main() -> int:
    loader = unittest.TestLoader()
    suite = loader.loadTestsFromModule(sys.modules[__name__])
    result = unittest.TextTestRunner(verbosity=2).run(suite)
    return 0 if result.wasSuccessful() else 1


if __name__ == "__main__":
    # Allow `python test_skills_orchestrator.py` without -m
    raise SystemExit(main())
