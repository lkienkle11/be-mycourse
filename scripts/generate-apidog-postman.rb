#!/usr/bin/env ruby
# frozen_string_literal: true

# Builds docs/api-dog-import.json — Postman Collection v2.1 (Apidog imports this).
# Tests (post-response) save ACCESS_TOKEN, REFRESH_TOKEN, SESSION_ID after
# login / confirm / refresh; SYSTEM_TOKEN after system login.
#
#   ruby scripts/generate-apidog-postman.rb

require "yaml"
require "json"
require "securerandom"

ROOT = File.expand_path("..", __dir__)
SPEC_PATH = File.join(ROOT, "docs", "api_swagger.yaml")
OUT_PATH = File.join(ROOT, "docs", "api-dog-import.json")

USER_TOKEN_SCRIPT = <<~'JS'.strip.lines.map(&:chomp)
  try {
    var json = pm.response.json();
    if (!json || json.code !== 0) { return; }
    var d = json.data;
    if (!d) { return; }
    function setBoth(key, val) {
      if (!val) { return; }
      try { pm.environment.set(key, val); } catch (e1) {}
      try { pm.collectionVariables.set(key, val); } catch (e2) {}
    }
    setBoth("ACCESS_TOKEN", d.access_token);
    setBoth("REFRESH_TOKEN", d.refresh_token);
    setBoth("SESSION_ID", d.session_id);
  } catch (e) {}
JS

SYSTEM_TOKEN_SCRIPT = <<~'JS'.strip.lines.map(&:chomp)
  try {
    var json = pm.response.json();
    if (!json || json.code !== 0) { return; }
    var d = json.data;
    if (d && d.access_token) {
      try { pm.environment.set("SYSTEM_TOKEN", d.access_token); } catch (e1) {}
      try { pm.collectionVariables.set("SYSTEM_TOKEN", d.access_token); } catch (e2) {}
    }
  } catch (e) {}
JS

def test_event(lines)
  {
    "listen" => "test",
    "script" => {
      "type" => "text/javascript",
      "exec" => lines.map { |l| "#{l}\n" }
    }
  }
end

def load_spec
  YAML.load_file(SPEC_PATH)
end

def resolve_ref(spec, ref)
  return nil unless ref.is_a?(String) && ref.start_with?("#/")
  ref.sub("#/", "").split("/").reduce(spec) { |obj, key| obj&.dig(key) }
end

def resolve_param(spec, p)
  p["$ref"] ? resolve_ref(spec, p["$ref"]) : p
end

def dummy_json_value(sc, spec)
  return nil unless sc.is_a?(Hash)
  sc = resolve_ref(spec, sc["$ref"]) if sc["$ref"]
  return "string" unless sc.is_a?(Hash)

  case sc["type"]
  when "string"
    (sc["enum"] && sc["enum"].first) || "string"
  when "integer", "number"
    1
  when "boolean"
    false
  when "array"
    []
  when "object"
    {}
  else
    "string"
  end
end

def json_example_from_schema(spec, schema)
  schema = resolve_ref(spec, schema["$ref"]) if schema.is_a?(Hash) && schema["$ref"]
  return "{}" unless schema.is_a?(Hash) && schema["properties"].is_a?(Hash)

  obj = {}
  required = schema["required"] || []
  schema["properties"].each do |key, sc|
    next unless required.include?(key)

    obj[key] = dummy_json_value(sc, spec)
  end
  JSON.pretty_generate(obj)
end

def merge_parameters(spec, path_item, op)
  list = []
  (path_item["parameters"] || []).each { |p| list << resolve_param(spec, p) }
  (op["parameters"] || []).each { |p| list << resolve_param(spec, p) }
  list.compact
end

def postman_url(path, query_list)
  raw = path.gsub(/\{([^}]+)\}/, '{{\1}}')
  raw = "{{BASE_URL}}#{raw}"
  unless query_list.empty?
    q = query_list.reject { |x| x["disabled"] }.map { |x| "#{x['key']}=#{x['value']}" }.join("&")
    raw = "#{raw}?#{q}" unless q.empty?
  end
  {
    "raw" => raw,
    "host" => ["{{BASE_URL}}"],
    "path" => path.sub(%r{\A/}, "").split("/").map { |seg| seg.gsub(/\{([^}]+)\}/, '{{\1}}') }
  }
end

def security_headers(op)
  sec = op["security"]
  h = []
  return h if sec.nil? || sec == []

  schemes = sec.flat_map(&:keys)
  if schemes.include?("bearerJwt")
    h << { "key" => "Authorization", "value" => "Bearer {{ACCESS_TOKEN}}", "type" => "text" }
  end
  if schemes.include?("systemBearer")
    h << { "key" => "Authorization", "value" => "Bearer {{SYSTEM_TOKEN}}", "type" => "text" }
  end
  if schemes.include?("internalApiKey")
    h << { "key" => "X-API-Key", "value" => "{{INTERNAL_KEY}}", "type" => "text" }
  end
  h
end

def header_params_to_headers(params)
  params.select { |p| p["in"] == "header" }.map do |p|
    name = p["name"]
    val = case name
          when "X-Refresh-Token" then "{{REFRESH_TOKEN}}"
          when "X-Session-Id" then "{{SESSION_ID}}"
          else "{{#{name.gsub(/[^a-zA-Z0-9]/, '_').upcase}}}"
          end
    { "key" => name, "value" => val, "type" => "text" }
  end
end

def query_params_to_urlencoded(params)
  params.select { |p| p["in"] == "query" }.map do |p|
    name = p["name"]
    ex = p.dig("schema", "example")
    val = ex || case name
                when "token" then "{{CONFIRM_TOKEN}}"
                when "page" then "1"
                when "per_page" then "20"
                else ""
                end
    { "key" => name, "value" => val.to_s, "disabled" => val.to_s.empty? }
  end
end

def build_body(spec, op)
  rb = op["requestBody"]
  return nil unless rb

  content = rb["content"] || {}
  if content["multipart/form-data"]
    return {
      "mode" => "formdata",
      "formdata" => [
        { "key" => "file", "type" => "file", "src" => "" },
        { "key" => "kind", "type" => "text", "value" => "FILE" },
        { "key" => "object_key", "type" => "text", "value" => "" },
        { "key" => "metadata", "type" => "text", "value" => "" }
      ]
    }
  end

  json_ct = content["application/json"]
  return nil unless json_ct

  schema = json_ct["schema"] || {}
  {
    "mode" => "raw",
    "raw" => json_example_from_schema(spec, schema),
    "options" => { "raw" => { "language" => "json" } }
  }
end

def token_events(method, path)
  case [method.upcase, path]
  when %w[POST /api/v1/auth/login], %w[GET /api/v1/auth/confirm], %w[POST /api/v1/auth/refresh]
    [test_event(USER_TOKEN_SCRIPT)]
  when %w[POST /api/system/login]
    [test_event(SYSTEM_TOKEN_SCRIPT)]
  else
    []
  end
end

def build_request(spec, path, method, path_item, op)
  params = merge_parameters(spec, path_item, op)
  headers = security_headers(op) + header_params_to_headers(params)
  if op["requestBody"]&.dig("content", "application/json")
    headers << { "key" => "Content-Type", "value" => "application/json", "type" => "text" }
  end

  name = op["summary"] || "#{method.upcase} #{path}"
  body = build_body(spec, op)

  q = query_params_to_urlencoded(params)
  req = {
    "method" => method.upcase,
    "header" => headers.uniq { |h| h["key"] },
    "url" => postman_url(path, q)
  }
  req["url"]["query"] = q unless q.empty?
  req["body"] = body if body

  item = { "name" => name, "request" => req }
  ev = token_events(method, path)
  item["event"] = ev unless ev.empty?
  item
end

def collect_items(spec)
  by_tag = Hash.new { |h, k| h[k] = [] }
  (spec["paths"] || {}).each do |path, path_item|
    next unless path_item.is_a?(Hash)

    path_item.each do |method, op|
      next unless op.is_a?(Hash) && op["responses"]

      tag = (op["tags"] || ["Default"]).first
      by_tag[tag] << build_request(spec, path, method, path_item, op)
    end
  end
  by_tag
end

def main
  spec = load_spec
  by_tag = collect_items(spec)
  order = (spec["tags"] || []).map { |t| t["name"] }

  folders = []
  order.each do |tag|
    items = by_tag[tag]
    folders << { "name" => tag, "item" => items } if items&.any?
  end
  (by_tag.keys - order).sort.each do |tag|
    folders << { "name" => tag, "item" => by_tag[tag] }
  end

  collection = {
    "info" => {
      "_postman_id" => SecureRandom.hex(11),
      "name" => spec.dig("info", "title") || "MyCourse API",
      "description" => <<~MD.strip,
        #{spec.dig("info", "description")}

        **Auto-save tokens (Tests tab):**
        - After **Login**, **Confirm email**, or **Refresh** → `ACCESS_TOKEN`, `REFRESH_TOKEN`, `SESSION_ID` are written to the **current Environment** and **Collection variables** (if `pm.environment.set` fails, collection vars still work).
        - After **System login** → `SYSTEM_TOKEN`.
        Create/select an **Environment** in Apidog with the same variable names for best UX.

        Regenerate this file: `ruby scripts/generate-apidog-postman.rb`
      MD
      "schema" => "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
    },
    "variable" => [
      { "key" => "BASE_URL", "value" => "http://localhost:8080" },
      { "key" => "ACCESS_TOKEN", "value" => "" },
      { "key" => "REFRESH_TOKEN", "value" => "" },
      { "key" => "SESSION_ID", "value" => "" },
      { "key" => "SYSTEM_TOKEN", "value" => "" },
      { "key" => "INTERNAL_KEY", "value" => "" },
      { "key" => "CONFIRM_TOKEN", "value" => "" }
    ],
    "item" => folders
  }

  File.write(OUT_PATH, JSON.pretty_generate(collection))
  warn "Wrote #{OUT_PATH} (#{File.size(OUT_PATH)} bytes)"
end

main
