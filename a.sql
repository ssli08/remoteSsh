CREATE TABLE `myproject` (
  `INSTANCE_NAME` varchar(100) NOT NULL DEFAULT '',
  `PUBLIC_IP` varchar(20) NOT NULL DEFAULT '',
  `PRIVATE_IP` varchar(20) NOT NULL DEFAULT '127.0.0.1',
  `REGION` varchar(100) NOT NULL DEFAULT '',
  `PROJECT` varchar(4) NOT NULL DEFAULT '',  	
  `ROLE` varchar(10),
  TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP); 
