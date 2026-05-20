#!/usr/bin/env ruby
# frozen_string_literal: true

# Builds docs/api-dog-import.json from docs/api_swagger.yaml (Postman Collection v2.1).
# All request/query examples MUST live in the OpenAPI spec — this script does not
# embed domain-specific sample payloads.
#
#   ruby scripts/generate-apidog-postman.rb

require "yaml"
require "json"
require "securerandom"

ROOT = File.expand_path("..", __dir__)
SPEC_PATH = File.join(ROOT, "docs", "api_swagger.yaml")
OUT_PATH = File.join(ROOT, "docs", "api-dog-import.json")

# Postman collection variable placeholders (not API payload examples).
HEADER_COLLECTION_VARS = {
  "X-Refresh-Token" => "{{REFRESH_TOKEN}}",
  "X-Session-Id" => "{{SESSION_ID}}"
}.freeze

def test_event(lines)
  {
    "listen" => "test",
    "script" => {
      "type" => "text/javascript",
      "exec" => lines.map { |l| "#{l}\n" }
    }
  }
end

def mycourse_config(spec)
  spec["x-mycourse"] || {}
end

def script_lines_from_exec(exec)
  case exec
  when Array
    exec.map { |l| l.to_s.chomp }
  when String
    exec.lines.map(&:chomp)
  else
    []
  end
end

def script_lines_for_key(spec, script_key)
  lines = mycourse_config(spec).dig("scriptLines", script_key)
  return script_lines_from_exec(lines) if lines

  nil
end

def post_response_events(spec, op)
  if op["x-postman-event"].is_a?(Array)
    op["x-postman-event"].map do |ev|
      next unless ev.is_a?(Hash) && ev["listen"] == "test"

      lines = script_lines_from_exec(ev.dig("script", "exec"))
      next if lines.empty?

      test_event(lines)
    end.compact
  elsif (key = op["x-mycourse-post-response"])
    lines = script_lines_for_key(spec, key)
    lines ? [test_event(lines)] : []
  else
    []
  end
end

def collection_variables(spec)
  env = mycourse_config(spec)["environment"] || {}
  env.map { |key, value| { "key" => key.to_s, "value" => value.to_s } }
end

def collection_auth(spec)
  auth = mycourse_config(spec)["collectionAuth"] || {}
  return nil unless auth["type"] == "bearer"

  {
    "type" => "bearer",
    "bearer" => [
      { "key" => "token", "value" => auth["token"].to_s, "type" => "string" }
    ]
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

def resolve_schema(spec, schema)
  return schema unless schema.is_a?(Hash)

  schema = resolve_ref(spec, schema["$ref"]) if schema["$ref"]
  schema
end

def example_value(spec, schema)
  schema = resolve_schema(spec, schema)
  return nil unless schema.is_a?(Hash)

  return schema["example"] if schema.key?("example")

  case schema["type"]
  when "string"
    schema["enum"]&.first || ""
  when "integer", "number"
    schema.fetch("default", 0)
  when "boolean"
    schema.fetch("default", false)
  when "array"
    []
  when "object"
    object_from_schema(spec, schema, required_only: true)
  else
    ""
  end
end

def object_from_schema(spec, schema, required_only: false)
  schema = resolve_schema(spec, schema)
  return schema["example"] if schema.is_a?(Hash) && schema.key?("example")
  return {} unless schema.is_a?(Hash) && schema["properties"].is_a?(Hash)

  required = schema["required"] || []
  obj = {}
  schema["properties"].each do |key, prop|
    prop = resolve_schema(spec, prop)
    if required_only && !required.include?(key) && !(prop.is_a?(Hash) && prop.key?("example"))
      next
    end

    obj[key] = example_value(spec, prop)
  end
  obj
end

def json_example_from_schema(spec, schema)
  schema = resolve_schema(spec, schema)
  return "{}" unless schema.is_a?(Hash)

  return JSON.pretty_generate(schema["example"]) if schema.key?("example")

  JSON.pretty_generate(object_from_schema(spec, schema, required_only: true))
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
    val = HEADER_COLLECTION_VARS[name] || query_or_schema_example(p).to_s
    { "key" => name, "value" => val, "type" => "text" }
  end
end

def query_or_schema_example(param)
  return param["example"] if param.key?("example")

  schema = param["schema"] || {}
  return schema["example"] if schema.key?("example")
  return schema["default"] if schema.key?("default")

  ""
end

def query_params_to_urlencoded(params)
  params.select { |p| p["in"] == "query" }.map do |p|
    val = query_or_schema_example(p).to_s
    { "key" => p["name"], "value" => val, "disabled" => val.empty? }
  end
end

def build_multipart_body(spec, schema)
  schema = resolve_schema(spec, schema)
  return nil unless schema.is_a?(Hash)

  if schema.key?("example") && schema["example"].is_a?(Hash)
    return {
      "mode" => "formdata",
      "formdata" => schema["example"].map do |key, val|
        if val.is_a?(Hash) && val["type"] == "file"
          { "key" => key, "type" => "file", "src" => val["src"].to_s }
        else
          { "key" => key, "type" => "text", "value" => val.to_s }
        end
      end
    }
  end

  required = schema["required"] || []
  formdata = []
  (schema["properties"] || {}).each do |key, prop|
    prop = resolve_schema(spec, prop)
    is_required = required.include?(key)
    if prop["format"] == "binary"
      formdata << { "key" => key, "type" => "file", "src" => "", "disabled" => !is_required }
    else
      val = example_value(spec, prop).to_s
      formdata << { "key" => key, "type" => "text", "value" => val, "disabled" => !is_required && val.empty? }
    end
  end
  { "mode" => "formdata", "formdata" => formdata }
end

def build_body(spec, op)
  rb = op["requestBody"]
  return nil unless rb

  content = rb["content"] || {}
  if content["multipart/form-data"]
    schema = content["multipart/form-data"]["schema"] || {}
    body = build_multipart_body(spec, schema)
    return body if body
  end

  json_ct = content["application/json"]
  return nil unless json_ct

  if json_ct.key?("example")
    return {
      "mode" => "raw",
      "raw" => JSON.pretty_generate(json_ct["example"]),
      "options" => { "raw" => { "language" => "json" } }
    }
  end

  schema = json_ct["schema"] || {}
  {
    "mode" => "raw",
    "raw" => json_example_from_schema(spec, schema),
    "options" => { "raw" => { "language" => "json" } }
  }
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
  ev = post_response_events(spec, op)
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

        Regenerate this file: `ruby scripts/generate-apidog-postman.rb` (token scripts + examples from `docs/api_swagger.yaml` only).
      MD
      "schema" => "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
    },
    "variable" => collection_variables(spec),
    "item" => folders
  }
  auth = collection_auth(spec)
  collection["auth"] = auth if auth

  File.write(OUT_PATH, JSON.pretty_generate(collection))
  warn "Wrote #{OUT_PATH} (#{File.size(OUT_PATH)} bytes)"
end

main
