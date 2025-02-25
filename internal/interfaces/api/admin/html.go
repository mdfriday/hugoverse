package admin

var startAdminHTML = `<!doctype html>
<html lang="en">
    <head>
        <title>{{ .Logo }}</title>
        <script type="text/javascript" src="/admin/static/common/js/jquery-2.1.4.min.js"></script>
        <script type="text/javascript" src="/admin/static/common/js/util.js"></script>
        <script type="text/javascript" src="/admin/static/dashboard/js/materialize.min.js"></script>
        <script type="text/javascript" src="/admin/static/dashboard/js/chart.bundle.min.js"></script>
        <script type="text/javascript" src="/admin/static/editor/js/materialNote.js"></script> 
        <script type="text/javascript" src="/admin/static/editor/js/ckMaterializeOverrides.js"></script>
                  
        <link rel="stylesheet" href="/admin/static/dashboard/css/material-icons.css" />     
        <link rel="stylesheet" href="/admin/static/dashboard/css/materialize.min.css" />
        <link rel="stylesheet" href="/admin/static/editor/css/materialNote.css" />
        <link rel="stylesheet" href="/admin/static/dashboard/css/admin.css" />    

        <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
        <meta charset="utf-8">
        <meta http-equiv="X-UA-Compatible" content="IE=edge">
    </head>
    <body class="grey lighten-4">
       <div class="navbar-fixed">
            <nav class="grey darken-2">
            <div class="nav-wrapper">
                <a class="brand-logo" href="/admin">{{ .Logo }}</a>

                <ul class="right">
                    <li><a href="/admin/logout">Logout</a></li>
                </ul>
            </div>
            </nav>
        </div>

        <div class="admin-ui row">`

var mainAdminHTML = `
            <div class="left-nav col s3">
                <div class="card">
                <ul class="card-content collection">
                    <div class="card-title">Content</div>
                                    
                    {{ range $t, $f := .Types }}
                    <div class="row collection-item">
                        <li><a class="col s12" href="/admin/contents?type={{ $t }}"><i class="tiny left material-icons">playlist_add</i>{{ $t }}</a></li>
                    </div>
                    {{ end }}

                    {{ if .IsAdmin }}
                    {{ range $t, $f := .AdminTypes }}
                    <div class="row collection-item">
                        <li><a class="col s12" href="/admin/contents?type={{ $t }}"><i class="tiny left material-icons">playlist_add</i>{{ $t }}</a></li>
                    </div>
                    {{ end }}

                    <div class="card-title">System</div>                                
                    <div class="row collection-item">
                        <li><a class="col s12" href="/admin/configure"><i class="tiny left material-icons">settings</i>Configuration</a></li>
                        <li><a class="col s12" href="/admin/configure/users"><i class="tiny left material-icons">supervisor_account</i>Users</a></li>
                        <li><a class="col s12" href="/admin/uploads"><i class="tiny left material-icons">swap_vert</i>Uploads</a></li>
                        <!-- <li><a class="col s12" href="/admin/addons"><i class="tiny left material-icons">settings_input_svideo</i>Addons</a></li> -->
                    </div>
                    {{ end }}

                </ul>
                </div>
            </div>
            {{ if .Subview}}
            <div class="subview col s9">
                {{ .Subview }}
            </div>
            {{ end }}`

var endAdminHTML = `
        </div>
        <footer class="row">
            <div class="col s12">
                <p class="center-align">
					Powered by &copy;<a target="_blank" href="https://gohugo.net">Hugoverse</a>
					&nbsp;&vert;&nbsp; 
					open-sourced by <a target="_blank" href="https://.mdfriday.com">MDFriday</a>
					&nbsp;&vert;&nbsp; 
					<a target="_blank" href="https://github.com/.mdfriday/hugoverse">GitHub</a>
				</p>
            </div>     
        </footer>
    </body>
</html>`

var initAdminHTML = `
<div class="init col s5">
<div class="card">
<div class="card-content">
    <div class="card-title">Welcome!</div>
    <blockquote>You need to initialize your system by filling out the form below. All of 
    this information can be updated later on, but you will not be able to start 
    without first completing this step.</blockquote>
    <form method="post" action="/admin/init" class="row">
        <div>Configuration</div>
        <div class="input-field col s12">        
            <input placeholder="Enter the name of your site (interal use only)" class="validate required" type="text" id="name" name="name"/>
            <label for="name" class="active">Site Namespace</label>
        </div>
        <div class="input-field col s12">        
            <input placeholder="Used for acquiring SSL certificate (e.g. www.example.com or  example.com)" class="validate" type="text" id="domain" name="domain"/>
            <label for="domain" class="active">Domain</label>
        </div>
        <div>Admin Details</div>
        <div class="input-field col s12">
            <input placeholder="Your email address e.g. you@example.com" class="validate required" type="email" id="email" name="email"/>
            <label for="email" class="active">Email</label>
        </div>
        <div class="input-field col s12">
            <input placeholder="Enter a strong password" class="validate required" type="password" id="password" name="password"/>
            <label for="password" class="active">Password</label>        
        </div>
        <button class="btn waves-effect waves-light right">Start</button>
    </form>
</div>
</div>
</div>
<script>
    $(function() {
        $('.nav-wrapper ul.right').hide();
        
        var logo = $('a.brand-logo');
        var name = $('input#name');
        var domain = $('input#domain');
        var hostname = domain.val();

        if (hostname === '') {    
            hostname = window.location.host || window.location.hostname;
        }
        
        if (hostname.indexOf(':') !== -1) {
            hostname = hostname.split(':')[0];
        }
        
        domain.val(hostname);
        
        name.on('change', function(e) {
            logo.text(e.target.value);
        });

    });
</script>
`

var analyticsHTML = `
<div class="analytics">
<div class="card">
<div class="card-content">
    <p class="right">Data range: {{ .from }} - {{ .to }} (UTC)</p>
    <div class="card-title">API Requests</div>
    <canvas id="analytics-chart"></canvas>
    <script>
    var target = document.getElementById("analytics-chart");
    Chart.defaults.global.defaultFontColor = '#212121';
    Chart.defaults.global.defaultFontFamily = "'Roboto', 'Helvetica Neue', 'Helvetica', 'Arial', 'sans-serif'";
    Chart.defaults.global.title.position = 'right';
    var chart = new Chart(target, {
        type: 'bar',
        data: {
            labels: [{{ range $date := .dates }} "{{ $date }}",  {{ end }}],
            datasets: [{
                type: 'line',
                label: 'Unique Clients',
                data: $.parseJSON({{ .unique }}),
                backgroundColor: 'rgba(76, 175, 80, 0.2)',
                borderColor: 'rgba(76, 175, 80, 1)',
                borderWidth: 1
            },
            {
                type: 'bar',
                label: 'Total Requests',
                data: $.parseJSON({{ .total }}),
                backgroundColor: 'rgba(33, 150, 243, 0.2)',
                borderColor: 'rgba(33, 150, 243, 1)',
                borderWidth: 1
            }]
        },
        options: {
            scales: {
                yAxes: [{
                    ticks: {
                        beginAtZero:true
                    }
                }]
            }
        }
    });
    </script>
</div>
</div>
</div>
`

var loginAdminHTML = `
<div class="init col s5">
<div class="card">
<div class="card-content">
    <div class="card-title">Welcome!</div>
    <blockquote>Please log in to the system using your email address and password.</blockquote>
    <form method="post" action="/admin/login" class="row">
        <div class="input-field col s12">
            <input placeholder="Enter your email address e.g. you@example.com" class="validate required" type="email" id="email" name="email"/>
            <label for="email" class="active">Email</label>
        </div>
        <div class="input-field col s12">
            <input placeholder="Enter your password" class="validate required" type="password" id="password" name="password"/>
            <a href="/admin/recover">Forgot password?</a>            
            <label for="password" class="active">Password</label>  
        </div>
        <button class="btn waves-effect waves-light right">Log in</button>
    </form>
</div>
</div>
</div>
<script>
    $(function() {
        $('.nav-wrapper ul.right').hide();
    });
</script>
`
