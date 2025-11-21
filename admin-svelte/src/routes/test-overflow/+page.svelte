<h1>测试内容溢出页面</h1>

<div class="overflow-content">
  {#each Array(50) as _, i}
    <div class="content-block">
      <h2>测试区块 {i + 1}</h2>
      <p>这是测试内容，用于模拟页面内容垂直溢出的情况。</p>
      <p>这是更多的测试文本，确保内容足够长以触发溢出行为。</p>
    </div>
  {/each}
</div>

<style>
  .overflow-content {
    width: 100%;
    box-sizing: border-box;
  }
  
  .content-block {
    margin-bottom: 2rem;
    padding: 1rem;
    background-color: #f3f4f6;
    border-radius: 8px;
  }
  
  h1 {
    color: #111827;
    margin-bottom: 2rem;
  }
</style>