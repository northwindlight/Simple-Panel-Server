var ws = new WebSocket('ws://192.168.175.107:60000');
var 处理器=document.getElementById("处理器");
var 温度=document.getElementById("温度");
var 内存=document.getElementById("内存");
var 存储=document.getElementById("存储");
var 处理器参数=document.getElementById("处理器参数");
var 温度参数=document.getElementById("温度参数");
var 内存参数=document.getElementById("内存参数");
var 存储参数=document.getElementById("存储参数");
var 处理器角度=document.getElementById("处理器角度");
var 温度角度=document.getElementById("温度角度");
var 内存角度=document.getElementById("内存角度");
var 存储角度 = document.getElementById("存储角度");
var 处理器频率 = document.getElementById("处理器频率");
var 温度描述 = document.getElementById("温度描述");
var 内存空间 = document.getElementById("内存空间");
var 磁盘空间 = document.getElementById("磁盘空间");
var connection = document.getElementById("connection");
var repeat = setInterval(ask, 8000);
var result = [];
const 百分号 = "<small>%</small>";
var 处理器数据 = 0;
var 温度数据 = 0;
var 内存数据 = 0;
var 存储数据 = 0;
var lasti = 0;
var lastj = 0;
var lastk = 0;
var lastl = 0;
function tempd(x) {
	if (x > 60) {
		return "温度较高";
	}
	else return "温度正常";
}
    function loop() {
        处理器参数.innerHTML = parseInt(处理器数据 / 100) + 百分号;
		温度参数.innerHTML = parseInt(温度数据 / 100) + "<small>℃</small>";
		内存参数.innerHTML = parseInt(内存数据 / 100) + 百分号;
		存储参数.innerHTML = parseInt(存储数据 / 100) + 百分号;
        let a = 处理器数据 * 0.036;
		let b = 温度数据 * 0.036;
		let c = 内存数据 * 0.036;
		let d = 存储数据 * 0.036;
        处理器.style.background = `conic-gradient(#1E40AE 0deg, #1E40AE ${a}deg, #E2E8F0 ${a}deg, #E2E8F0 360deg)`;
        处理器角度.style.transform = `rotate(${a}deg)`;
		温度.style.background = `conic-gradient(#1E40AE 0deg, #1E40AE ${b}deg, #E2E8F0 ${b}deg, #E2E8F0 360deg)`;
        温度角度.style.transform = `rotate(${b}deg)`;
		内存.style.background = `conic-gradient(#1E40AE 0deg, #1E40AE ${c}deg, #E2E8F0 ${c}deg, #E2E8F0 360deg)`;
        内存角度.style.transform = `rotate(${c}deg)`;
		存储.style.background = `conic-gradient(#1E40AE 0deg, #1E40AE ${d}deg, #E2E8F0 ${d}deg, #E2E8F0 360deg)`;
        存储角度.style.transform = `rotate(${d}deg)`;
		if (处理器数据 != result[0] * 100)
			{
				处理器数据+=i ;
			}
		if (温度数据 != result[1] * 100)
			{
				温度数据+=j;
			}
		if (内存数据 != result[2] * 100)
			{
				内存数据+=k;
			}
		if (存储数据 != result[3] * 100)
			{
				存储数据+=l;
			}
            window.requestAnimationFrame(loop);
		return;
    }
ws.addEventListener('open', function (event) {

	setTimeout(function() {
  // 这里是延迟执行的代码
		connection.innerHTML = "CONNECTED"; 
}, 400);
		connection.style.backgroundColor = "green";
		connection.style.width = "120px";
    ws.send('Ask message!');
});

// 监听动态 数据
ws.addEventListener('message', function (event) {
	str = event.data;
	console.log(str);
	result = [];
	处理器数据 = lasti;
	温度数据 = lastj;
	内存数据 = lastk;
	存储数据 = lastl;
	for (var z = 0; z < str.length; z += 3) {
		if (z + 2 >= str.length) {
			break; // 如果已经到达最后两位了，就不再切割了
		}
		result.push(parseInt(str.slice(z, z + 3),10)); 
	}
	console.log(result);
	i = (result[0] * 100 - 处理器数据) / 100;
	j = (result[1] * 100 - 温度数据) / 100;
	k = (result[2] * 100 - 内存数据) / 100;
	l = (result[3] * 100 - 存储数据) / 100;
	lasti = result[0] * 100;
	lastj = result[1] * 100;
	lastk = result[2] * 100;
	lastl = result[3] * 100;
	setTimeout(function () {
		// 这里是延迟执行的代码
		处理器频率.innerHTML = result[4] + "Mhz";
		温度描述.innerHTML = tempd(result[1]);
		内存空间.innerHTML = result[6] + "MB / " + result[5] + "MB";
		磁盘空间.innerHTML = result[8] + "GB / " + result[7] + "GB";
	}, 450);
	loop()

});


ws.addEventListener("close", function(event) {
	setTimeout(function() {
  // 这里是延迟执行的代码
		connection.innerHTML = "CLOSED"; 
}, 300);
		connection.style.backgroundColor = "red";
		connection.style.width = "80px";
		clearInterval(repeat);	
		console.log("closed");
  // handle close event
});


function ask(){
	if(ws.readyState == WebSocket.OPEN){
		ws.send('Hello Server!');
		console.log("again");
		return;
}
}


