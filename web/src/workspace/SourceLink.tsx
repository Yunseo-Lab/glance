const sourceRef =
  import.meta.env.VITE_SOURCE_REF || "release/personal-dashboard";
export default function SourceLink() {
  const url =
    "https://github.com/Yunseo-Lab/glance/tree/" +
    sourceRef.split("/").map(encodeURIComponent).join("/");
  return (
    <a href={url} target="_blank" rel="noopener noreferrer">
      소스 코드 · AGPL-3.0
    </a>
  );
}
